package domain

import "time"

type SiteDiscoveryRunStatus string

const (
	SiteDiscoveryRunStatusRunning   SiteDiscoveryRunStatus = "running"
	SiteDiscoveryRunStatusSucceeded SiteDiscoveryRunStatus = "succeeded"
	SiteDiscoveryRunStatusFailed    SiteDiscoveryRunStatus = "failed"
)

type SiteDiscoveryClassificationStatus string

const (
	SiteDiscoveryClassificationSupported   SiteDiscoveryClassificationStatus = "supported"
	SiteDiscoveryClassificationUnknown     SiteDiscoveryClassificationStatus = "unknown"
	SiteDiscoveryClassificationUnsupported SiteDiscoveryClassificationStatus = "unsupported"
)

type SiteDiscoveryImportStatus string

const (
	SiteDiscoveryImportNew      SiteDiscoveryImportStatus = "new"
	SiteDiscoveryImportImported SiteDiscoveryImportStatus = "imported"
	SiteDiscoveryImportSkipped  SiteDiscoveryImportStatus = "skipped"
)

type SiteDiscoveryProcessStatus string

const (
	SiteDiscoveryProcessUnprocessed    SiteDiscoveryProcessStatus = "unprocessed"
	SiteDiscoveryProcessAddedToCatalog SiteDiscoveryProcessStatus = "added_to_catalog"
	SiteDiscoveryProcessRegistered     SiteDiscoveryProcessStatus = "registered"
	SiteDiscoveryProcessIgnored        SiteDiscoveryProcessStatus = "ignored"
)

type SupplierRegistrationStatus string

const (
	SupplierRegistrationStatusPending                   SupplierRegistrationStatus = "pending"
	SupplierRegistrationStatusQueued                    SupplierRegistrationStatus = "queued"
	SupplierRegistrationStatusRunning                   SupplierRegistrationStatus = "running"
	SupplierRegistrationStatusWaitingManualVerification SupplierRegistrationStatus = "waiting_manual_verification"
	SupplierRegistrationStatusSucceeded                 SupplierRegistrationStatus = "succeeded"
	SupplierRegistrationStatusFailed                    SupplierRegistrationStatus = "failed"
)

type SiteDiscoverySettings struct {
	RegistrationEmail   string    `json:"registration_email"`
	RegistrationEnabled bool      `json:"registration_enabled"`
	LowRateThreshold    float64   `json:"low_rate_threshold"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type SiteDiscoveryRun struct {
	ID             int64                  `json:"id"`
	SourceURL      string                 `json:"source_url"`
	Status         SiteDiscoveryRunStatus `json:"status"`
	Total          int                    `json:"total"`
	SupportedTotal int                    `json:"supported_total"`
	ImportedTotal  int                    `json:"imported_total"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	StartedAt      time.Time              `json:"started_at"`
	FinishedAt     *time.Time             `json:"finished_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type SiteDiscoveryItem struct {
	ID                       int64                             `json:"id"`
	RunID                    int64                             `json:"run_id"`
	SourceURL                string                            `json:"source_url"`
	SourceSiteID             string                            `json:"source_site_id"`
	SourceSection            string                            `json:"source_section"`
	SourceCategory           string                            `json:"source_category,omitempty"`
	Name                     string                            `json:"name"`
	RegisterURL              string                            `json:"register_url"`
	DashboardURL             string                            `json:"dashboard_url"`
	APIBaseURL               string                            `json:"api_base_url"`
	Host                     string                            `json:"host"`
	DomainHint               string                            `json:"domain_hint,omitempty"`
	Description              string                            `json:"description,omitempty"`
	ProviderType             SupplierType                      `json:"provider_type,omitempty"`
	ClassificationStatus     SiteDiscoveryClassificationStatus `json:"classification_status"`
	ClassificationConfidence float64                           `json:"classification_confidence"`
	ClassificationEvidence   []string                          `json:"classification_evidence,omitempty"`
	MonitorStatus            string                            `json:"monitor_status,omitempty"`
	MonitorAvailable         *bool                             `json:"monitor_available,omitempty"`
	MonitorUptimePercent     *float64                          `json:"monitor_uptime_percent,omitempty"`
	MonitorAvgResponseMS     *int                              `json:"monitor_avg_response_ms,omitempty"`
	MonitorLatestResponseMS  *int                              `json:"monitor_latest_response_ms,omitempty"`
	ImportStatus             SiteDiscoveryImportStatus         `json:"import_status"`
	ProcessStatus            SiteDiscoveryProcessStatus        `json:"process_status"`
	CatalogSiteID            int64                             `json:"catalog_site_id,omitempty"`
	SupplierID               int64                             `json:"supplier_id,omitempty"`
	RegistrationStatus       SupplierRegistrationStatus        `json:"registration_status,omitempty"`
	RegistrationTaskID       int64                             `json:"registration_task_id,omitempty"`
	RegistrationEmail        string                            `json:"registration_email,omitempty"`
	RegistrationErrorCode    string                            `json:"registration_error_code,omitempty"`
	RegistrationErrorMessage string                            `json:"registration_error_message,omitempty"`
	RawPayload               map[string]any                    `json:"raw_payload,omitempty"`
	CreatedAt                time.Time                         `json:"created_at"`
	UpdatedAt                time.Time                         `json:"updated_at"`
}

type SupplierRegistrationCredential struct {
	ID                 int64                      `json:"id"`
	DiscoveryID        int64                      `json:"discovery_id"`
	SupplierID         int64                      `json:"supplier_id"`
	Email              string                     `json:"email"`
	PasswordCiphertext string                     `json:"-"`
	PasswordConfigured bool                       `json:"password_configured"`
	Status             SupplierRegistrationStatus `json:"status"`
	VerificationStatus string                     `json:"verification_status,omitempty"`
	ExtensionTaskID    int64                      `json:"extension_task_id,omitempty"`
	ErrorCode          string                     `json:"error_code,omitempty"`
	ErrorMessage       string                     `json:"error_message,omitempty"`
	LastAttemptAt      *time.Time                 `json:"last_attempt_at,omitempty"`
	CreatedAt          time.Time                  `json:"created_at"`
	UpdatedAt          time.Time                  `json:"updated_at"`
}

type SiteDiscoveryRecommendation struct {
	Item                *SiteDiscoveryItem `json:"item"`
	MinRateMultiplier   float64            `json:"min_rate_multiplier"`
	RecommendedChannels int                `json:"recommended_channels"`
	Reason              string             `json:"reason"`
}
