package kanban

import (
	"context"
	"net/url"
	"strings"

	sitecatalogapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/sitecatalog"
	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

const defaultPriceSourceDiscoveryLimit = 50

type SiteCatalogReader interface {
	ListSites(ctx context.Context, filter sitecatalogapp.SiteFilter) ([]*adminplusdomain.SiteCatalogSite, error)
}

type PriceSourceDiscoveryInput struct {
	Query                string
	Limit                int
	IncludeLowConfidence bool
}

type MarketPriceSourceCandidate struct {
	SiteID     int64          `json:"site_id,omitempty"`
	SupplierID int64          `json:"supplier_id,omitempty"`
	SourceType string         `json:"source_type"`
	SourceName string         `json:"source_name"`
	SourceURL  string         `json:"source_url"`
	LinkType   string         `json:"link_type,omitempty"`
	Confidence float64        `json:"confidence"`
	Reason     string         `json:"reason"`
	RawPayload map[string]any `json:"raw_payload,omitempty"`
}

type PriceSourceDiscoveryResult struct {
	Items []*MarketPriceSourceCandidate `json:"items"`
	Total int                           `json:"total"`
}

func (s *Service) DiscoverMarketPriceSources(ctx context.Context, in PriceSourceDiscoveryInput) (*PriceSourceDiscoveryResult, error) {
	if s == nil || s.siteCatalog == nil {
		return nil, internalError("site catalog reader is not configured")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultPriceSourceDiscoveryLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}
	sites, err := s.siteCatalog.ListSites(ctx, sitecatalogapp.SiteFilter{
		Query: strings.TrimSpace(in.Query),
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	items := discoverPriceSourceCandidates(sites, limit, in.IncludeLowConfidence)
	return &PriceSourceDiscoveryResult{Items: items, Total: len(items)}, nil
}

func discoverPriceSourceCandidates(sites []*adminplusdomain.SiteCatalogSite, limit int, includeLowConfidence bool) []*MarketPriceSourceCandidate {
	items := make([]*MarketPriceSourceCandidate, 0, limit)
	seen := map[string]struct{}{}
	for _, site := range sites {
		if site == nil {
			continue
		}
		for _, link := range site.Links {
			if len(items) >= limit {
				return items
			}
			if link == nil {
				continue
			}
			candidate := candidateFromSiteLink(site, link)
			if candidate == nil || shouldSkipPriceSourceCandidate(candidate, includeLowConfidence, seen) {
				continue
			}
			seen[candidate.SourceURL] = struct{}{}
			items = append(items, candidate)
		}
		for _, candidate := range derivedPriceSourceCandidates(site) {
			if len(items) >= limit {
				return items
			}
			if shouldSkipPriceSourceCandidate(candidate, includeLowConfidence, seen) {
				continue
			}
			seen[candidate.SourceURL] = struct{}{}
			items = append(items, candidate)
		}
	}
	return items
}

func candidateFromSiteLink(site *adminplusdomain.SiteCatalogSite, link *adminplusdomain.SiteCatalogLink) *MarketPriceSourceCandidate {
	sourceURL := normalizeCandidateURL(link.URL)
	if sourceURL == "" {
		return nil
	}
	confidence, reason := scorePriceSourceLink(link)
	return &MarketPriceSourceCandidate{
		SiteID:     site.ID,
		SupplierID: site.SupplierID,
		SourceType: "site_catalog",
		SourceName: sourceNameFromSite(site),
		SourceURL:  sourceURL,
		LinkType:   string(link.LinkType),
		Confidence: confidence,
		Reason:     reason,
		RawPayload: map[string]any{
			"site_slug":    site.Slug,
			"site_host":    site.CanonicalHost,
			"link_label":   link.Label,
			"link_type":    string(link.LinkType),
			"link_status":  string(link.Status),
			"discovery_by": "site_catalog_link",
		},
	}
}

func derivedPriceSourceCandidates(site *adminplusdomain.SiteCatalogSite) []*MarketPriceSourceCandidate {
	origin := siteOrigin(site)
	if origin == "" {
		return nil
	}
	paths := []struct {
		path       string
		confidence float64
		reason     string
	}{
		{path: "/pricing", confidence: 0.62, reason: "derived_pricing_path"},
		{path: "/price", confidence: 0.58, reason: "derived_price_path"},
		{path: "/models", confidence: 0.5, reason: "derived_models_path"},
		{path: "/recharge", confidence: 0.46, reason: "derived_recharge_path"},
	}
	items := make([]*MarketPriceSourceCandidate, 0, len(paths))
	for _, item := range paths {
		items = append(items, &MarketPriceSourceCandidate{
			SiteID:     site.ID,
			SupplierID: site.SupplierID,
			SourceType: "site_catalog",
			SourceName: sourceNameFromSite(site),
			SourceURL:  origin + item.path,
			LinkType:   "derived",
			Confidence: item.confidence,
			Reason:     item.reason,
			RawPayload: map[string]any{
				"site_slug":    site.Slug,
				"site_host":    site.CanonicalHost,
				"discovery_by": "derived_common_path",
			},
		})
	}
	return items
}

func shouldSkipPriceSourceCandidate(candidate *MarketPriceSourceCandidate, includeLowConfidence bool, seen map[string]struct{}) bool {
	if candidate == nil || candidate.SourceURL == "" {
		return true
	}
	if _, ok := seen[candidate.SourceURL]; ok {
		return true
	}
	return !includeLowConfidence && candidate.Confidence < 0.55
}

func scorePriceSourceLink(link *adminplusdomain.SiteCatalogLink) (float64, string) {
	haystack := strings.ToLower(link.URL + " " + link.Label + " " + string(link.LinkType))
	switch {
	case containsAny(haystack, "pricing", "prices", "price", "费用", "价格", "套餐", "计费", "billing", "倍率", "rate"):
		return 0.92, "price_keyword_in_link"
	case link.LinkType == adminplusdomain.SiteCatalogLinkRecharge || containsAny(haystack, "recharge", "topup", "充值"):
		return 0.74, "recharge_or_topup_link"
	case link.LinkType == adminplusdomain.SiteCatalogLinkDocs || containsAny(haystack, "docs", "document", "文档"):
		return 0.56, "docs_link_may_contain_pricing"
	case link.LinkType == adminplusdomain.SiteCatalogLinkHomepage:
		return 0.48, "homepage_candidate"
	default:
		return 0.4, "low_confidence_link"
	}
}

func siteOrigin(site *adminplusdomain.SiteCatalogSite) string {
	for _, link := range site.Links {
		if link == nil || link.URL == "" {
			continue
		}
		if link.LinkType == adminplusdomain.SiteCatalogLinkHomepage || link.IsPrimary {
			if origin := originFromURL(link.URL); origin != "" {
				return origin
			}
		}
	}
	if site.CanonicalHost == "" {
		return ""
	}
	host := strings.TrimSpace(site.CanonicalHost)
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	return originFromURL(host)
}

func originFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func normalizeCandidateURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Fragment = ""
	return parsed.String()
}

func sourceNameFromSite(site *adminplusdomain.SiteCatalogSite) string {
	return firstNonEmpty(site.ShortName, site.Name, site.CanonicalHost, site.Slug)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
