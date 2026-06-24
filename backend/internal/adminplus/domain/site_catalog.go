package domain

import "time"

type SiteCatalogStatus string

const (
	SiteCatalogStatusDraft     SiteCatalogStatus = "draft"
	SiteCatalogStatusReviewing SiteCatalogStatus = "reviewing"
	SiteCatalogStatusPublished SiteCatalogStatus = "published"
	SiteCatalogStatusArchived  SiteCatalogStatus = "archived"
)

type SiteCatalogVisibility string

const (
	SiteCatalogVisibilityPublic  SiteCatalogVisibility = "public"
	SiteCatalogVisibilityPrivate SiteCatalogVisibility = "private"
)

type SiteCatalogQualityStatus string

const (
	SiteCatalogQualityComplete    SiteCatalogQualityStatus = "complete"
	SiteCatalogQualityNeedsReview SiteCatalogQualityStatus = "needs_review"
	SiteCatalogQualityLinkBroken  SiteCatalogQualityStatus = "link_broken"
	SiteCatalogQualityDuplicate   SiteCatalogQualityStatus = "duplicate"
)

type SiteCatalogKind string

const (
	SiteCatalogKindAPIRelay  SiteCatalogKind = "api_relay"
	SiteCatalogKindOfficial  SiteCatalogKind = "official"
	SiteCatalogKindTool      SiteCatalogKind = "tool"
	SiteCatalogKindClient    SiteCatalogKind = "client"
	SiteCatalogKindBenchmark SiteCatalogKind = "benchmark"
	SiteCatalogKindOther     SiteCatalogKind = "other"
)

type SiteCatalogRecommendationLevel string

const (
	SiteCatalogRecommendationNone     SiteCatalogRecommendationLevel = "none"
	SiteCatalogRecommendationNormal   SiteCatalogRecommendationLevel = "normal"
	SiteCatalogRecommendationFeatured SiteCatalogRecommendationLevel = "featured"
	SiteCatalogRecommendationAvoid    SiteCatalogRecommendationLevel = "avoid"
)

type SiteCatalogRiskLevel string

const (
	SiteCatalogRiskUnknown SiteCatalogRiskLevel = "unknown"
	SiteCatalogRiskLow     SiteCatalogRiskLevel = "low"
	SiteCatalogRiskMedium  SiteCatalogRiskLevel = "medium"
	SiteCatalogRiskHigh    SiteCatalogRiskLevel = "high"
)

type SiteCatalogLinkType string

const (
	SiteCatalogLinkHomepage  SiteCatalogLinkType = "homepage"
	SiteCatalogLinkRegister  SiteCatalogLinkType = "register"
	SiteCatalogLinkDashboard SiteCatalogLinkType = "dashboard"
	SiteCatalogLinkAPIBase   SiteCatalogLinkType = "api_base"
	SiteCatalogLinkRecharge  SiteCatalogLinkType = "recharge"
	SiteCatalogLinkDocs      SiteCatalogLinkType = "docs"
	SiteCatalogLinkContact   SiteCatalogLinkType = "contact"
)

type SiteCatalogLinkStatus string

const (
	SiteCatalogLinkUnknown SiteCatalogLinkStatus = "unknown"
	SiteCatalogLinkOK      SiteCatalogLinkStatus = "ok"
	SiteCatalogLinkBroken  SiteCatalogLinkStatus = "broken"
)

type SiteCatalogSite struct {
	ID                   int64                          `json:"id"`
	Slug                 string                         `json:"slug"`
	CanonicalHost        string                         `json:"canonical_host"`
	Name                 string                         `json:"name"`
	ShortName            string                         `json:"short_name,omitempty"`
	Summary              string                         `json:"summary,omitempty"`
	Description          string                         `json:"description,omitempty"`
	ProviderType         SupplierType                   `json:"provider_type,omitempty"`
	SiteKind             SiteCatalogKind                `json:"site_kind"`
	Status               SiteCatalogStatus              `json:"status"`
	Visibility           SiteCatalogVisibility          `json:"visibility"`
	QualityStatus        SiteCatalogQualityStatus       `json:"quality_status"`
	RecommendationLevel  SiteCatalogRecommendationLevel `json:"recommendation_level"`
	RecommendationReason string                         `json:"recommendation_reason,omitempty"`
	RiskLevel            SiteCatalogRiskLevel           `json:"risk_level"`
	LogoURL              string                         `json:"logo_url,omitempty"`
	ScreenshotURL        string                         `json:"screenshot_url,omitempty"`
	PrimaryLanguage      string                         `json:"primary_language,omitempty"`
	CountryOrRegion      string                         `json:"country_or_region,omitempty"`
	SupplierID           int64                          `json:"supplier_id,omitempty"`
	Metadata             map[string]any                 `json:"metadata,omitempty"`
	Links                []*SiteCatalogLink             `json:"links,omitempty"`
	Categories           []*SiteCatalogCategory         `json:"categories,omitempty"`
	Tags                 []*SiteCatalogTag              `json:"tags,omitempty"`
	Sources              []*SiteCatalogSource           `json:"sources,omitempty"`
	PublishedAt          *time.Time                     `json:"published_at,omitempty"`
	CreatedAt            time.Time                      `json:"created_at"`
	UpdatedAt            time.Time                      `json:"updated_at"`
}

type SiteCatalogLink struct {
	ID            int64                 `json:"id"`
	SiteID        int64                 `json:"site_id"`
	LinkType      SiteCatalogLinkType   `json:"link_type"`
	URL           string                `json:"url"`
	Label         string                `json:"label,omitempty"`
	IsPrimary     bool                  `json:"is_primary"`
	Status        SiteCatalogLinkStatus `json:"status"`
	LastCheckedAt *time.Time            `json:"last_checked_at,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

type SiteCatalogCategory struct {
	ID           int64     `json:"id"`
	ParentID     int64     `json:"parent_id,omitempty"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	DisplayOrder int       `json:"display_order"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SiteCatalogTag struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	TagType   string    `json:"tag_type"`
	Color     string    `json:"color,omitempty"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SiteCatalogSource struct {
	ID                   int64          `json:"id"`
	SiteID               int64          `json:"site_id"`
	SourceType           string         `json:"source_type"`
	SourceName           string         `json:"source_name"`
	SourceURL            string         `json:"source_url,omitempty"`
	SourceExternalID     string         `json:"source_external_id,omitempty"`
	DiscoveryCandidateID int64          `json:"discovery_candidate_id,omitempty"`
	ObservedPayload      map[string]any `json:"observed_payload,omitempty"`
	FirstSeenAt          time.Time      `json:"first_seen_at"`
	LastSeenAt           time.Time      `json:"last_seen_at"`
	CreatedAt            time.Time      `json:"created_at"`
}
