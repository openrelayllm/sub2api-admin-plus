package suppliers

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type CreateSupplierInput struct {
	Name                string
	Kind                adminplusdomain.SupplierKind
	Type                adminplusdomain.SupplierType
	RuntimeStatus       adminplusdomain.SupplierRuntimeStatus
	HealthStatus        adminplusdomain.SupplierHealthStatus
	DashboardURL        string
	APIBaseURL          string
	Contact             string
	Notes               string
	AdminAPIKey         string
	PostgresReadDSN     string
	RedisReadDSN        string
	BrowserLoginEnabled bool
	BalanceCents        int64
	BalanceCurrency     string
}

type UpdateSupplierStatusInput struct {
	RuntimeStatus adminplusdomain.SupplierRuntimeStatus
	HealthStatus  adminplusdomain.SupplierHealthStatus
}

type SupplierFilter struct {
	Kind          adminplusdomain.SupplierKind
	Type          adminplusdomain.SupplierType
	RuntimeStatus adminplusdomain.SupplierRuntimeStatus
	HealthStatus  adminplusdomain.SupplierHealthStatus
	Query         string
}

type Repository interface {
	Create(ctx context.Context, supplier *adminplusdomain.Supplier) (*adminplusdomain.Supplier, error)
	Get(ctx context.Context, id int64) (*adminplusdomain.Supplier, error)
	List(ctx context.Context, filter SupplierFilter) ([]*adminplusdomain.Supplier, error)
	UpdateStatus(ctx context.Context, id int64, runtimeStatus adminplusdomain.SupplierRuntimeStatus, healthStatus adminplusdomain.SupplierHealthStatus) (*adminplusdomain.Supplier, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) Create(ctx context.Context, in CreateSupplierInput) (*adminplusdomain.Supplier, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("supplier service is not configured")
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, badRequest("SUPPLIER_NAME_REQUIRED", "supplier name is required")
	}
	if len(name) > 80 {
		return nil, badRequest("SUPPLIER_NAME_TOO_LONG", "supplier name must be 80 characters or less")
	}
	if !in.Kind.Valid() {
		return nil, badRequest("SUPPLIER_KIND_INVALID", "invalid supplier kind")
	}
	if !in.Type.Valid() {
		return nil, badRequest("SUPPLIER_TYPE_INVALID", "invalid supplier type")
	}
	runtimeStatus := in.RuntimeStatus
	if runtimeStatus == "" {
		runtimeStatus = adminplusdomain.SupplierRuntimeStatusMonitorOnly
	}
	if !runtimeStatus.Valid() {
		return nil, badRequest("SUPPLIER_RUNTIME_STATUS_INVALID", "invalid supplier runtime status")
	}
	healthStatus := in.HealthStatus
	if healthStatus == "" {
		healthStatus = adminplusdomain.SupplierHealthStatusNormal
	}
	if !healthStatus.Valid() {
		return nil, badRequest("SUPPLIER_HEALTH_STATUS_INVALID", "invalid supplier health status")
	}
	if runtimeStatus == adminplusdomain.SupplierRuntimeStatusCandidate && in.BalanceCents <= 0 {
		return nil, badRequest("SUPPLIER_BALANCE_REQUIRED_FOR_CANDIDATE", "candidate supplier must have positive balance")
	}
	if runtimeStatus == adminplusdomain.SupplierRuntimeStatusActive && in.BalanceCents <= 0 {
		return nil, badRequest("SUPPLIER_BALANCE_REQUIRED_FOR_ACTIVE", "active supplier must have positive balance")
	}
	dashboardURL, err := normalizeOptionalURL(in.DashboardURL, "SUPPLIER_DASHBOARD_URL_INVALID")
	if err != nil {
		return nil, err
	}
	apiBaseURL, err := normalizeOptionalURL(in.APIBaseURL, "SUPPLIER_API_BASE_URL_INVALID")
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	supplier := &adminplusdomain.Supplier{
		Name:          name,
		Kind:          in.Kind,
		Type:          in.Type,
		RuntimeStatus: runtimeStatus,
		HealthStatus:  healthStatus,
		DashboardURL:  dashboardURL,
		APIBaseURL:    apiBaseURL,
		Contact:       trimLimit(in.Contact, 120),
		Notes:         trimLimit(in.Notes, 500),
		Credential: adminplusdomain.SupplierCredentialStatus{
			AdminAPIKeyConfigured: strings.TrimSpace(in.AdminAPIKey) != "",
			PostgresConfigured:    strings.TrimSpace(in.PostgresReadDSN) != "",
			RedisConfigured:       strings.TrimSpace(in.RedisReadDSN) != "",
			BrowserLoginEnabled:   in.BrowserLoginEnabled,
			MaskedAdminAPIKey:     maskSecret(in.AdminAPIKey),
		},
		BalanceCents:    in.BalanceCents,
		BalanceCurrency: normalizeCurrency(in.BalanceCurrency),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if in.BalanceCents > 0 {
		t := now
		supplier.BalanceUpdatedAt = &t
	}
	return s.repo.Create(ctx, supplier)
}

func (s *Service) Get(ctx context.Context, id int64) (*adminplusdomain.Supplier, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("supplier service is not configured")
	}
	if id <= 0 {
		return nil, badRequest("SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	return s.repo.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, filter SupplierFilter) ([]*adminplusdomain.Supplier, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("supplier service is not configured")
	}
	if filter.Kind != "" && !filter.Kind.Valid() {
		return nil, badRequest("SUPPLIER_KIND_INVALID", "invalid supplier kind")
	}
	if filter.Type != "" && !filter.Type.Valid() {
		return nil, badRequest("SUPPLIER_TYPE_INVALID", "invalid supplier type")
	}
	if filter.RuntimeStatus != "" && !filter.RuntimeStatus.Valid() {
		return nil, badRequest("SUPPLIER_RUNTIME_STATUS_INVALID", "invalid supplier runtime status")
	}
	if filter.HealthStatus != "" && !filter.HealthStatus.Valid() {
		return nil, badRequest("SUPPLIER_HEALTH_STATUS_INVALID", "invalid supplier health status")
	}
	filter.Query = strings.ToLower(strings.TrimSpace(filter.Query))
	return s.repo.List(ctx, filter)
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, in UpdateSupplierStatusInput) (*adminplusdomain.Supplier, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("supplier service is not configured")
	}
	if id <= 0 {
		return nil, badRequest("SUPPLIER_ID_INVALID", "invalid supplier id")
	}
	if !in.RuntimeStatus.Valid() {
		return nil, badRequest("SUPPLIER_RUNTIME_STATUS_INVALID", "invalid supplier runtime status")
	}
	if !in.HealthStatus.Valid() {
		return nil, badRequest("SUPPLIER_HEALTH_STATUS_INVALID", "invalid supplier health status")
	}
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.RuntimeStatus == adminplusdomain.SupplierRuntimeStatusCandidate && existing.BalanceCents <= 0 {
		return nil, badRequest("SUPPLIER_BALANCE_REQUIRED_FOR_CANDIDATE", "candidate supplier must have positive balance")
	}
	if in.RuntimeStatus == adminplusdomain.SupplierRuntimeStatusActive && existing.BalanceCents <= 0 {
		return nil, badRequest("SUPPLIER_BALANCE_REQUIRED_FOR_ACTIVE", "active supplier must have positive balance")
	}
	return s.repo.UpdateStatus(ctx, id, in.RuntimeStatus, in.HealthStatus)
}

func normalizeOptionalURL(raw string, reason string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", nil
	}
	u, err := url.ParseRequestURI(v)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", badRequest(reason, "invalid supplier url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", badRequest(reason, "supplier url must use http or https")
	}
	return v, nil
}

func trimLimit(value string, limit int) string {
	v := strings.TrimSpace(value)
	if len(v) <= limit {
		return v
	}
	return v[:limit]
}

func normalizeCurrency(value string) string {
	v := strings.ToUpper(strings.TrimSpace(value))
	if v == "" {
		return "USD"
	}
	if len(v) != 3 {
		return "USD"
	}
	return v
}

func maskSecret(value string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if len(v) <= 8 {
		return "****"
	}
	return v[:4] + "..." + v[len(v)-4:]
}

func badRequest(reason string, message string) error {
	return infraerrors.New(http.StatusBadRequest, reason, message)
}

func internalError(message string) error {
	return infraerrors.New(http.StatusInternalServerError, "ADMIN_PLUS_INTERNAL_ERROR", message)
}
