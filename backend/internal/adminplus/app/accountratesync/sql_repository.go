package accountratesync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	sub2apiapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/sub2api"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type SQLRepository struct {
	db      *sql.DB
	sub2api *sql.DB
}

func NewSQLRepository(db *sql.DB, sub2apiReadDB sub2apiapp.ReadDB) *SQLRepository {
	readDB := sub2apiReadDB.DB
	if readDB == nil {
		readDB = db
	}
	return &SQLRepository{db: db, sub2api: readDB}
}

func (r *SQLRepository) ListLocalAPIKeyAccounts(ctx context.Context, platform string, limit int) ([]*LocalAccount, error) {
	if r == nil || r.sub2api == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "SUB2API_READ_DB_NOT_CONFIGURED", "sub2api read database is not configured")
	}
	platform = protocolToPlatform(platform)
	rows, err := r.sub2api.QueryContext(ctx, `
		SELECT id, name, platform, type, status, schedulable, credentials, extra, updated_at
		FROM accounts
		WHERE deleted_at IS NULL
			AND platform = $1
			AND type = $2
		ORDER BY id DESC
		LIMIT $3
	`, platform, service.AccountTypeAPIKey, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*LocalAccount, 0)
	for rows.Next() {
		item, err := scanLocalAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLRepository) GetLocalAccount(ctx context.Context, accountID int64) (*LocalAccount, error) {
	if r == nil || r.sub2api == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "SUB2API_READ_DB_NOT_CONFIGURED", "sub2api read database is not configured")
	}
	row := r.sub2api.QueryRowContext(ctx, `
		SELECT id, name, platform, type, status, schedulable, credentials, extra, updated_at
		FROM accounts
		WHERE id = $1 AND deleted_at IS NULL
	`, accountID)
	account, err := scanLocalAccount(row)
	if err == sql.ErrNoRows {
		return nil, infraerrors.New(http.StatusNotFound, "LOCAL_ACCOUNT_NOT_FOUND", "local account not found")
	}
	return account, err
}

func (r *SQLRepository) FindMatchesByMetadata(ctx context.Context, supplierID int64, supplierKeyID int64, supplierGroupID int64, externalGroupID string, platform string) ([]*Match, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	if supplierID <= 0 && supplierKeyID <= 0 && supplierGroupID <= 0 && strings.TrimSpace(externalGroupID) == "" {
		return []*Match{}, nil
	}
	where := []string{"sk.status = 'bound'", "sg.status = 'active'", "sg.effective_rate_multiplier > 0"}
	args := make([]any, 0, 5)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	if supplierID > 0 {
		where = append(where, "s.id = "+addArg(supplierID))
	}
	if supplierKeyID > 0 {
		where = append(where, "sk.id = "+addArg(supplierKeyID))
	}
	if supplierGroupID > 0 {
		where = append(where, "sg.id = "+addArg(supplierGroupID))
	}
	if strings.TrimSpace(externalGroupID) != "" {
		where = append(where, "sg.external_group_id = "+addArg(strings.TrimSpace(externalGroupID)))
	}
	return r.findMatches(ctx, strings.Join(where, " AND "), args, platform, "metadata")
}

func (r *SQLRepository) FindMatchesByFingerprint(ctx context.Context, fingerprint string, platform string) ([]*Match, error) {
	if strings.TrimSpace(fingerprint) == "" {
		return []*Match{}, nil
	}
	return r.findMatches(ctx, "sk.status = 'bound' AND sg.status = 'active' AND sg.effective_rate_multiplier > 0 AND sk.key_fingerprint = $1", []any{strings.TrimSpace(fingerprint)}, platform, "fingerprint")
}

func (r *SQLRepository) FindMatchesByLast4(ctx context.Context, last4 string, platform string) ([]*Match, error) {
	if strings.TrimSpace(last4) == "" {
		return []*Match{}, nil
	}
	return r.findMatches(ctx, "sk.status = 'bound' AND sg.status = 'active' AND sg.effective_rate_multiplier > 0 AND sk.key_last4 = $1", []any{strings.TrimSpace(last4)}, platform, "last4")
}

func (r *SQLRepository) findMatches(ctx context.Context, where string, args []any, platform string, source string) ([]*Match, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			s.id,
			s.name,
			s.type,
			sg.id,
			sg.name,
			sk.id,
			sg.effective_rate_multiplier,
			sg.provider_family,
			COALESCE(sg.description, ''),
			COALESCE(sg.official_name, ''),
			COALESCE(sg.model_family, ''),
			COALESCE(sg.model_spec, '')
		FROM admin_plus_supplier_keys sk
		INNER JOIN admin_plus_supplier_groups sg ON sg.id = sk.supplier_group_id AND sg.supplier_id = sk.supplier_id
		INNER JOIN admin_plus_suppliers s ON s.id = sk.supplier_id
		WHERE `+where+`
			AND s.runtime_status <> 'disabled'
		ORDER BY sg.effective_rate_multiplier ASC, sk.id ASC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]*Match, 0)
	for rows.Next() {
		var item Match
		var providerFamily, description, officialName, modelFamily, modelSpec string
		if err := rows.Scan(
			&item.SupplierID,
			&item.SupplierName,
			&item.SupplierType,
			&item.SupplierGroupID,
			&item.SupplierGroupName,
			&item.SupplierKeyID,
			&item.EffectiveRateMultiplier,
			&providerFamily,
			&description,
			&officialName,
			&modelFamily,
			&modelSpec,
		); err != nil {
			return nil, err
		}
		if !matchPlatform(platform, providerFamily, item.SupplierGroupName, description, officialName, modelFamily, modelSpec) {
			continue
		}
		item.MatchSource = source
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SQLRepository) ListLatestHistory(ctx context.Context, accountIDs []int64) (map[int64]*adminplusdomain.AccountRateSyncHistory, error) {
	out := make(map[int64]*adminplusdomain.AccountRateSyncHistory, len(accountIDs))
	if len(accountIDs) == 0 {
		return out, nil
	}
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT ON (local_sub2api_account_id)
			id, local_sub2api_account_id, local_account_name, local_account_platform,
			key_fingerprint, key_last4, supplier_id, supplier_name, supplier_type,
			supplier_group_id, supplier_group_name, supplier_key_id, match_source,
			effective_rate_multiplier, target_account_name, status, error_code,
			error_message, renamed, old_account_name, new_account_name, synced_at, created_at
		FROM admin_plus_account_rate_sync_history
		WHERE local_sub2api_account_id = ANY($1)
		ORDER BY local_sub2api_account_id, synced_at DESC, id DESC
	`, pq.Array(accountIDs))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		item, err := scanHistory(rows)
		if err != nil {
			return nil, err
		}
		out[item.LocalSub2APIAccountID] = item
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SQLRepository) CreateHistory(ctx context.Context, history *adminplusdomain.AccountRateSyncHistory) (*adminplusdomain.AccountRateSyncHistory, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	if history == nil {
		return nil, infraerrors.New(http.StatusBadRequest, "ACCOUNT_RATE_SYNC_HISTORY_REQUIRED", "account rate sync history is required")
	}
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO admin_plus_account_rate_sync_history (
			local_sub2api_account_id, local_account_name, local_account_platform,
			key_fingerprint, key_last4, supplier_id, supplier_name, supplier_type,
			supplier_group_id, supplier_group_name, supplier_key_id, match_source,
			effective_rate_multiplier, target_account_name, status, error_code,
			error_message, renamed, old_account_name, new_account_name, synced_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id, local_sub2api_account_id, local_account_name, local_account_platform,
			key_fingerprint, key_last4, supplier_id, supplier_name, supplier_type,
			supplier_group_id, supplier_group_name, supplier_key_id, match_source,
			effective_rate_multiplier, target_account_name, status, error_code,
			error_message, renamed, old_account_name, new_account_name, synced_at, created_at
	`,
		history.LocalSub2APIAccountID,
		history.LocalAccountName,
		history.LocalAccountPlatform,
		history.KeyFingerprint,
		history.KeyLast4,
		history.SupplierID,
		history.SupplierName,
		history.SupplierType,
		history.SupplierGroupID,
		history.SupplierGroupName,
		history.SupplierKeyID,
		history.MatchSource,
		history.EffectiveRateMultiplier,
		history.TargetAccountName,
		history.Status,
		history.ErrorCode,
		history.ErrorMessage,
		history.Renamed,
		history.OldAccountName,
		history.NewAccountName,
		history.SyncedAt,
		history.CreatedAt,
	)
	return scanHistory(row)
}

func (r *SQLRepository) GetHistory(ctx context.Context, id int64) (*adminplusdomain.AccountRateSyncHistory, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, local_sub2api_account_id, local_account_name, local_account_platform,
			key_fingerprint, key_last4, supplier_id, supplier_name, supplier_type,
			supplier_group_id, supplier_group_name, supplier_key_id, match_source,
			effective_rate_multiplier, target_account_name, status, error_code,
			error_message, renamed, old_account_name, new_account_name, synced_at, created_at
		FROM admin_plus_account_rate_sync_history
		WHERE id = $1
	`, id)
	item, err := scanHistory(row)
	if err == sql.ErrNoRows {
		return nil, infraerrors.New(http.StatusNotFound, "ACCOUNT_RATE_SYNC_HISTORY_NOT_FOUND", "account rate sync history not found")
	}
	return item, err
}

func (r *SQLRepository) MarkRenamed(ctx context.Context, id int64, oldName string, newName string, targetName string) (*adminplusdomain.AccountRateSyncHistory, error) {
	if r == nil || r.db == nil {
		return nil, dbNotConfigured()
	}
	row := r.db.QueryRowContext(ctx, `
		UPDATE admin_plus_account_rate_sync_history
		SET status = 'renamed',
			renamed = TRUE,
			old_account_name = $2,
			new_account_name = $3,
			target_account_name = $4
		WHERE id = $1
		RETURNING id, local_sub2api_account_id, local_account_name, local_account_platform,
			key_fingerprint, key_last4, supplier_id, supplier_name, supplier_type,
			supplier_group_id, supplier_group_name, supplier_key_id, match_source,
			effective_rate_multiplier, target_account_name, status, error_code,
			error_message, renamed, old_account_name, new_account_name, synced_at, created_at
	`, id, oldName, newName, targetName)
	item, err := scanHistory(row)
	if err == sql.ErrNoRows {
		return nil, infraerrors.New(http.StatusNotFound, "ACCOUNT_RATE_SYNC_HISTORY_NOT_FOUND", "account rate sync history not found")
	}
	return item, err
}

func (r *SQLRepository) ClearHistory(ctx context.Context) (int64, error) {
	if r == nil || r.db == nil {
		return 0, dbNotConfigured()
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM admin_plus_account_rate_sync_history`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type localAccountScanner interface {
	Scan(dest ...any) error
}

func scanLocalAccount(scanner localAccountScanner) (*LocalAccount, error) {
	var item LocalAccount
	var credentialsRaw, extraRaw []byte
	if err := scanner.Scan(
		&item.ID,
		&item.Name,
		&item.Platform,
		&item.Type,
		&item.Status,
		&item.Schedulable,
		&credentialsRaw,
		&extraRaw,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Credentials = decodeJSONMap(credentialsRaw)
	item.Extra = decodeJSONMap(extraRaw)
	return &item, nil
}

type historyScanner interface {
	Scan(dest ...any) error
}

func scanHistory(scanner historyScanner) (*adminplusdomain.AccountRateSyncHistory, error) {
	var item adminplusdomain.AccountRateSyncHistory
	var status string
	if err := scanner.Scan(
		&item.ID,
		&item.LocalSub2APIAccountID,
		&item.LocalAccountName,
		&item.LocalAccountPlatform,
		&item.KeyFingerprint,
		&item.KeyLast4,
		&item.SupplierID,
		&item.SupplierName,
		&item.SupplierType,
		&item.SupplierGroupID,
		&item.SupplierGroupName,
		&item.SupplierKeyID,
		&item.MatchSource,
		&item.EffectiveRateMultiplier,
		&item.TargetAccountName,
		&status,
		&item.ErrorCode,
		&item.ErrorMessage,
		&item.Renamed,
		&item.OldAccountName,
		&item.NewAccountName,
		&item.SyncedAt,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.Status = adminplusdomain.AccountRateSyncStatus(status)
	return &item, nil
}

func decodeJSONMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func matchPlatform(platform string, values ...string) bool {
	platform = protocolToPlatform(platform)
	text := strings.ToLower(strings.Join(values, " "))
	switch platform {
	case service.PlatformAnthropic:
		return strings.Contains(text, "anthropic") || strings.Contains(text, "claude")
	default:
		return strings.Contains(text, "openai") || strings.Contains(text, "gpt")
	}
}

func dbNotConfigured() error {
	return infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_DB_NOT_CONFIGURED", "admin plus database is not configured")
}
