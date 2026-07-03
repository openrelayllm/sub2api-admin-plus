package accountratesync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const (
	defaultAccountLimit = 500
	maxAccountLimit     = 2000
)

type LocalAccount struct {
	ID          int64
	Name        string
	Platform    string
	Type        string
	Status      string
	Schedulable bool
	Credentials map[string]any
	Extra       map[string]any
	UpdatedAt   time.Time
}

type Match struct {
	SupplierID              int64
	SupplierName            string
	SupplierType            string
	SupplierGroupID         int64
	SupplierGroupName       string
	SupplierKeyID           int64
	EffectiveRateMultiplier float64
	MatchSource             string
}

type Row struct {
	LocalSub2APIAccountID   int64                                   `json:"local_sub2api_account_id"`
	LocalAccountName        string                                  `json:"local_account_name"`
	LocalAccountPlatform    string                                  `json:"local_account_platform"`
	LocalAccountStatus      string                                  `json:"local_account_status,omitempty"`
	LocalAccountSchedulable bool                                    `json:"local_account_schedulable"`
	KeyLast4                string                                  `json:"key_last4,omitempty"`
	History                 *adminplusdomain.AccountRateSyncHistory `json:"history,omitempty"`
	Status                  adminplusdomain.AccountRateSyncStatus   `json:"status"`
	SupplierID              int64                                   `json:"supplier_id,omitempty"`
	SupplierName            string                                  `json:"supplier_name,omitempty"`
	SupplierType            string                                  `json:"supplier_type,omitempty"`
	SupplierGroupID         int64                                   `json:"supplier_group_id,omitempty"`
	SupplierGroupName       string                                  `json:"supplier_group_name,omitempty"`
	SupplierKeyID           int64                                   `json:"supplier_key_id,omitempty"`
	MatchSource             string                                  `json:"match_source,omitempty"`
	EffectiveRateMultiplier float64                                 `json:"effective_rate_multiplier"`
	TargetAccountName       string                                  `json:"target_account_name,omitempty"`
	Renamed                 bool                                    `json:"renamed"`
	ErrorCode               string                                  `json:"error_code,omitempty"`
	ErrorMessage            string                                  `json:"error_message,omitempty"`
	SyncedAt                *time.Time                              `json:"synced_at,omitempty"`
}

type ListResult struct {
	Items []*Row `json:"items"`
	Total int    `json:"total"`
}

type SyncResult struct {
	Items     []*Row `json:"items"`
	Total     int    `json:"total"`
	Matched   int    `json:"matched"`
	Renamed   int    `json:"renamed"`
	NotFound  int    `json:"not_found"`
	Ambiguous int    `json:"ambiguous"`
	Failed    int    `json:"failed"`
}

type Repository interface {
	ListLocalAPIKeyAccounts(ctx context.Context, platform string, limit int) ([]*LocalAccount, error)
	GetLocalAccount(ctx context.Context, accountID int64) (*LocalAccount, error)
	FindMatchesByMetadata(ctx context.Context, supplierID int64, supplierKeyID int64, supplierGroupID int64, externalGroupID string, platform string) ([]*Match, error)
	FindMatchesByFingerprint(ctx context.Context, fingerprint string, platform string) ([]*Match, error)
	FindMatchesByLast4(ctx context.Context, last4 string, platform string) ([]*Match, error)
	ListLatestHistory(ctx context.Context, accountIDs []int64) (map[int64]*adminplusdomain.AccountRateSyncHistory, error)
	CreateHistory(ctx context.Context, history *adminplusdomain.AccountRateSyncHistory) (*adminplusdomain.AccountRateSyncHistory, error)
	GetHistory(ctx context.Context, id int64) (*adminplusdomain.AccountRateSyncHistory, error)
	MarkRenamed(ctx context.Context, id int64, oldName string, newName string, targetName string) (*adminplusdomain.AccountRateSyncHistory, error)
	ClearHistory(ctx context.Context) (int64, error)
}

type AccountUpdater interface {
	UpdateAccount(ctx context.Context, id int64, input *service.UpdateAccountInput) (*service.Account, error)
}

type Service struct {
	repo    Repository
	updater AccountUpdater
	now     func() time.Time
}

func NewService(repo Repository, updater service.AdminService) *Service {
	return &Service{repo: repo, updater: updater, now: time.Now}
}

func (s *Service) List(ctx context.Context, protocol string, limit int) (*ListResult, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("account rate sync service is not configured")
	}
	accounts, err := s.repo.ListLocalAPIKeyAccounts(ctx, protocolToPlatform(protocol), normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		if account != nil {
			ids = append(ids, account.ID)
		}
	}
	latest, err := s.repo.ListLatestHistory(ctx, ids)
	if err != nil {
		return nil, err
	}
	rows := make([]*Row, 0, len(accounts))
	for _, account := range accounts {
		rows = append(rows, rowFromAccount(account, latest[account.ID]))
	}
	return &ListResult{Items: rows, Total: len(rows)}, nil
}

func (s *Service) Sync(ctx context.Context, protocol string, limit int) (*SyncResult, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("account rate sync service is not configured")
	}
	accounts, err := s.repo.ListLocalAPIKeyAccounts(ctx, protocolToPlatform(protocol), normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	result := &SyncResult{Items: make([]*Row, 0, len(accounts))}
	for _, account := range accounts {
		row, err := s.syncLocalAccount(ctx, account)
		if err != nil {
			row = failedRow(account, "ACCOUNT_RATE_SYNC_FAILED", err.Error(), s.now().UTC())
		}
		result.Items = append(result.Items, row)
		addResultStats(result, row)
	}
	result.Total = len(result.Items)
	return result, nil
}

func (s *Service) RetryAccount(ctx context.Context, accountID int64) (*Row, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("account rate sync service is not configured")
	}
	if accountID <= 0 {
		return nil, badRequest("LOCAL_ACCOUNT_ID_INVALID", "invalid local account id")
	}
	account, err := s.repo.GetLocalAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return s.syncLocalAccount(ctx, account)
}

func (s *Service) RenameFromHistory(ctx context.Context, historyID int64) (*Row, error) {
	if s == nil || s.repo == nil || s.updater == nil {
		return nil, internalError("account rate sync rename service is not configured")
	}
	if historyID <= 0 {
		return nil, badRequest("ACCOUNT_RATE_SYNC_HISTORY_ID_INVALID", "invalid account rate sync history id")
	}
	history, err := s.repo.GetHistory(ctx, historyID)
	if err != nil {
		return nil, err
	}
	if history.LocalSub2APIAccountID <= 0 {
		return nil, conflict("LOCAL_ACCOUNT_ID_MISSING", "local account id is missing")
	}
	if history.TargetAccountName == "" {
		return nil, conflict("ACCOUNT_RATE_SYNC_TARGET_NAME_MISSING", "target account name is missing")
	}
	if history.Status != adminplusdomain.AccountRateSyncStatusMatched && history.Status != adminplusdomain.AccountRateSyncStatusRenamed {
		return nil, conflict("ACCOUNT_RATE_SYNC_NOT_MATCHED", "only matched account rate sync history can be renamed")
	}
	oldName := history.LocalAccountName
	updated, err := s.updater.UpdateAccount(ctx, history.LocalSub2APIAccountID, &service.UpdateAccountInput{Name: history.TargetAccountName})
	if err != nil {
		return nil, err
	}
	if updated != nil && updated.Name != "" {
		oldName = firstNonEmpty(history.LocalAccountName, updated.Name)
	}
	renamed, err := s.repo.MarkRenamed(ctx, history.ID, oldName, history.TargetAccountName, history.TargetAccountName)
	if err != nil {
		return nil, err
	}
	account, err := s.repo.GetLocalAccount(ctx, history.LocalSub2APIAccountID)
	if err != nil {
		return rowFromHistory(renamed), nil
	}
	return rowFromAccount(account, renamed), nil
}

func (s *Service) RenameMatched(ctx context.Context, protocol string, limit int) (*SyncResult, error) {
	list, err := s.List(ctx, protocol, limit)
	if err != nil {
		return nil, err
	}
	result := &SyncResult{Items: make([]*Row, 0, len(list.Items))}
	for _, row := range list.Items {
		if row == nil || row.History == nil || row.TargetAccountName == "" || row.Renamed {
			continue
		}
		if row.Status != adminplusdomain.AccountRateSyncStatusMatched {
			continue
		}
		renamed, err := s.RenameFromHistory(ctx, row.History.ID)
		if err != nil {
			row.Status = adminplusdomain.AccountRateSyncStatusFailed
			row.ErrorCode = "ACCOUNT_RENAME_FAILED"
			row.ErrorMessage = err.Error()
			result.Items = append(result.Items, row)
			addResultStats(result, row)
			continue
		}
		result.Items = append(result.Items, renamed)
		addResultStats(result, renamed)
	}
	result.Total = len(result.Items)
	return result, nil
}

func (s *Service) ClearHistory(ctx context.Context) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, internalError("account rate sync service is not configured")
	}
	return s.repo.ClearHistory(ctx)
}

func (s *Service) syncLocalAccount(ctx context.Context, account *LocalAccount) (*Row, error) {
	if account == nil {
		return nil, badRequest("LOCAL_ACCOUNT_REQUIRED", "local account is required")
	}
	now := s.now().UTC()
	secret := extractAPIKey(account.Credentials)
	fingerprint := fingerprintSecret(secret)
	last4 := lastN(secret, 4)
	base := &adminplusdomain.AccountRateSyncHistory{
		LocalSub2APIAccountID: account.ID,
		LocalAccountName:      account.Name,
		LocalAccountPlatform:  account.Platform,
		KeyFingerprint:        fingerprint,
		KeyLast4:              last4,
		Status:                adminplusdomain.AccountRateSyncStatusNotFound,
		SyncedAt:              now,
		CreatedAt:             now,
	}
	if secret == "" {
		base.Status = adminplusdomain.AccountRateSyncStatusFailed
		base.ErrorCode = "LOCAL_ACCOUNT_API_KEY_MISSING"
		base.ErrorMessage = "local account api key is missing"
		created, err := s.repo.CreateHistory(ctx, base)
		if err != nil {
			return nil, err
		}
		return rowFromAccount(account, created), nil
	}

	match, status, code, message, err := s.matchAccount(ctx, account, fingerprint, last4)
	if err != nil {
		return nil, err
	}
	base.Status = status
	base.ErrorCode = code
	base.ErrorMessage = message
	if match != nil {
		base.SupplierID = match.SupplierID
		base.SupplierName = match.SupplierName
		base.SupplierType = match.SupplierType
		base.SupplierGroupID = match.SupplierGroupID
		base.SupplierGroupName = match.SupplierGroupName
		base.SupplierKeyID = match.SupplierKeyID
		base.MatchSource = match.MatchSource
		base.EffectiveRateMultiplier = match.EffectiveRateMultiplier
		base.TargetAccountName = targetAccountName(match.SupplierName, match.EffectiveRateMultiplier)
	}
	created, err := s.repo.CreateHistory(ctx, base)
	if err != nil {
		return nil, err
	}
	return rowFromAccount(account, created), nil
}

func (s *Service) matchAccount(ctx context.Context, account *LocalAccount, fingerprint string, last4 string) (*Match, adminplusdomain.AccountRateSyncStatus, string, string, error) {
	platform := protocolToPlatform(account.Platform)
	metadataMatches, err := s.repo.FindMatchesByMetadata(ctx,
		int64FromMap(account.Extra, "admin_plus_supplier_id"),
		int64FromMap(account.Extra, "admin_plus_supplier_key"),
		int64FromMap(account.Extra, "admin_plus_supplier_group_id"),
		stringFromMap(account.Extra, "admin_plus_external_group_id"),
		platform,
	)
	if err != nil {
		return nil, "", "", "", err
	}
	if selected, status, code, message := chooseMatch(metadataMatches, "metadata"); selected != nil || status == adminplusdomain.AccountRateSyncStatusAmbiguous {
		return selected, status, code, message, nil
	}

	if fingerprint != "" {
		fingerprintMatches, err := s.repo.FindMatchesByFingerprint(ctx, fingerprint, platform)
		if err != nil {
			return nil, "", "", "", err
		}
		if selected, status, code, message := chooseMatch(fingerprintMatches, "fingerprint"); selected != nil || status == adminplusdomain.AccountRateSyncStatusAmbiguous {
			return selected, status, code, message, nil
		}
	}
	if last4 != "" {
		last4Matches, err := s.repo.FindMatchesByLast4(ctx, last4, platform)
		if err != nil {
			return nil, "", "", "", err
		}
		if selected, status, code, message := chooseMatch(last4Matches, "last4"); selected != nil || status == adminplusdomain.AccountRateSyncStatusAmbiguous {
			return selected, status, code, message, nil
		}
	}
	return nil, adminplusdomain.AccountRateSyncStatusNotFound, "SUPPLIER_KEY_MATCH_NOT_FOUND", "supplier key match not found; sync supplier groups/keys first", nil
}

func chooseMatch(matches []*Match, source string) (*Match, adminplusdomain.AccountRateSyncStatus, string, string) {
	candidates := make([]*Match, 0, len(matches))
	for _, match := range matches {
		if match != nil && match.EffectiveRateMultiplier > 0 {
			cp := *match
			cp.MatchSource = source
			candidates = append(candidates, &cp)
		}
	}
	if len(candidates) == 0 {
		return nil, adminplusdomain.AccountRateSyncStatusNotFound, "", ""
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].EffectiveRateMultiplier != candidates[j].EffectiveRateMultiplier {
			return candidates[i].EffectiveRateMultiplier < candidates[j].EffectiveRateMultiplier
		}
		return candidates[i].SupplierKeyID < candidates[j].SupplierKeyID
	})
	if len(candidates) > 1 {
		first := candidates[0]
		for _, candidate := range candidates[1:] {
			if candidate.SupplierKeyID != first.SupplierKeyID || candidate.SupplierGroupID != first.SupplierGroupID || candidate.SupplierID != first.SupplierID {
				return nil, adminplusdomain.AccountRateSyncStatusAmbiguous, "SUPPLIER_KEY_MATCH_AMBIGUOUS", fmt.Sprintf("matched %d supplier keys; please repair binding first", len(candidates))
			}
		}
	}
	return candidates[0], adminplusdomain.AccountRateSyncStatusMatched, "", ""
}

func rowFromAccount(account *LocalAccount, history *adminplusdomain.AccountRateSyncHistory) *Row {
	row := &Row{
		LocalSub2APIAccountID:   account.ID,
		LocalAccountName:        account.Name,
		LocalAccountPlatform:    account.Platform,
		LocalAccountStatus:      account.Status,
		LocalAccountSchedulable: account.Schedulable,
		KeyLast4:                lastN(extractAPIKey(account.Credentials), 4),
		Status:                  adminplusdomain.AccountRateSyncStatusNotFound,
	}
	if history != nil {
		applyHistory(row, history)
	}
	return row
}

func rowFromHistory(history *adminplusdomain.AccountRateSyncHistory) *Row {
	row := &Row{}
	applyHistory(row, history)
	return row
}

func applyHistory(row *Row, history *adminplusdomain.AccountRateSyncHistory) {
	if row == nil || history == nil {
		return
	}
	row.History = history
	row.LocalSub2APIAccountID = firstPositive(row.LocalSub2APIAccountID, history.LocalSub2APIAccountID)
	row.LocalAccountName = firstNonEmpty(row.LocalAccountName, history.LocalAccountName)
	row.LocalAccountPlatform = firstNonEmpty(row.LocalAccountPlatform, history.LocalAccountPlatform)
	row.KeyLast4 = firstNonEmpty(row.KeyLast4, history.KeyLast4)
	row.Status = history.Status
	row.SupplierID = history.SupplierID
	row.SupplierName = history.SupplierName
	row.SupplierType = history.SupplierType
	row.SupplierGroupID = history.SupplierGroupID
	row.SupplierGroupName = history.SupplierGroupName
	row.SupplierKeyID = history.SupplierKeyID
	row.MatchSource = history.MatchSource
	row.EffectiveRateMultiplier = history.EffectiveRateMultiplier
	row.TargetAccountName = history.TargetAccountName
	row.Renamed = history.Renamed
	row.ErrorCode = history.ErrorCode
	row.ErrorMessage = history.ErrorMessage
	syncedAt := history.SyncedAt
	row.SyncedAt = &syncedAt
}

func failedRow(account *LocalAccount, code string, message string, syncedAt time.Time) *Row {
	history := &adminplusdomain.AccountRateSyncHistory{
		Status:       adminplusdomain.AccountRateSyncStatusFailed,
		ErrorCode:    code,
		ErrorMessage: message,
		SyncedAt:     syncedAt,
		CreatedAt:    syncedAt,
	}
	if account != nil {
		history.LocalSub2APIAccountID = account.ID
		history.LocalAccountName = account.Name
		history.LocalAccountPlatform = account.Platform
		history.KeyLast4 = lastN(extractAPIKey(account.Credentials), 4)
		return rowFromAccount(account, history)
	}
	return rowFromHistory(history)
}

func addResultStats(result *SyncResult, row *Row) {
	if result == nil || row == nil {
		return
	}
	switch row.Status {
	case adminplusdomain.AccountRateSyncStatusMatched:
		result.Matched++
	case adminplusdomain.AccountRateSyncStatusRenamed:
		result.Matched++
		result.Renamed++
	case adminplusdomain.AccountRateSyncStatusNotFound:
		result.NotFound++
	case adminplusdomain.AccountRateSyncStatusAmbiguous:
		result.Ambiguous++
	case adminplusdomain.AccountRateSyncStatusFailed:
		result.Failed++
	}
}

func targetAccountName(supplierName string, rate float64) string {
	name := strings.TrimSpace(supplierName)
	if name == "" || rate <= 0 {
		return ""
	}
	return trimLimit(name+"-"+formatMultiplier(rate), 120)
}

func formatMultiplier(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64) + "x"
}

func extractAPIKey(credentials map[string]any) string {
	for _, key := range []string{"api_key", "apiKey", "apikey", "key", "token", "secret"} {
		if value := stringFromMap(credentials, key); value != "" {
			return value
		}
	}
	return ""
}

func fingerprintSecret(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func lastN(value string, n int) string {
	value = strings.TrimSpace(value)
	if len(value) <= n {
		return value
	}
	return value[len(value)-n:]
}

func protocolToPlatform(protocol string) string {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "claude", service.PlatformAnthropic:
		return service.PlatformAnthropic
	default:
		return service.PlatformOpenAI
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultAccountLimit
	}
	if limit > maxAccountLimit {
		return maxAccountLimit
	}
	return limit
}

func stringFromMap(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	switch v := values[key].(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func int64FromMap(values map[string]any, key string) int64 {
	if len(values) == 0 {
		return 0
	}
	switch v := values[key].(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return n
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositive(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func trimLimit(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit > 0 && len(value) > limit {
		return value[:limit]
	}
	return value
}

func internalError(message string) error {
	return infraerrors.New(http.StatusInternalServerError, "ACCOUNT_RATE_SYNC_INTERNAL_ERROR", message)
}

func badRequest(reason string, message string) error {
	return infraerrors.New(http.StatusBadRequest, reason, message)
}

func conflict(reason string, message string) error {
	return infraerrors.New(http.StatusConflict, reason, message)
}
