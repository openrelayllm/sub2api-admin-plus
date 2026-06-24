package sitediscovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
)

func TestParseDaheiAIItems(t *testing.T) {
	body := `
		<section id="third-party">
			<a class="card" href="https://example.com/register" data-site-id="site-1" data-domain="example.com">
				<div class="name">Example New API</div>
				<div class="desc">new-api 模板渠道</div>
			</a>
		</section>
	`
	items, err := parseDaheiAIItems(DefaultSourceURL, body)
	if err != nil {
		t.Fatalf("parse items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	if item.SourceSiteID != "site-1" {
		t.Fatalf("unexpected site id: %q", item.SourceSiteID)
	}
	if item.SourceSection != "third-party" {
		t.Fatalf("unexpected section: %q", item.SourceSection)
	}
	if item.RegisterURL != "https://example.com/register" {
		t.Fatalf("unexpected register url: %q", item.RegisterURL)
	}
	if item.APIBaseURL != "https://example.com" {
		t.Fatalf("unexpected api base url: %q", item.APIBaseURL)
	}
}

func TestClassifyItem(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected adminplusdomain.SupplierType
	}{
		{name: "new api", text: "new-api 模板 支持 New-Api-User", expected: adminplusdomain.SupplierTypeNewAPI},
		{name: "sub2api", text: "sub2api admin channel", expected: adminplusdomain.SupplierTypeSub2API},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyItem(&adminplusdomain.SiteDiscoveryItem{
				Name:        tt.text,
				Description: tt.text,
			})
			if result.Status != adminplusdomain.SiteDiscoveryClassificationSupported {
				t.Fatalf("expected supported, got %s", result.Status)
			}
			if result.ProviderType != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result.ProviderType)
			}
		})
	}

	unknown := classifyItem(&adminplusdomain.SiteDiscoveryItem{Name: "plain site"})
	if unknown.Status != adminplusdomain.SiteDiscoveryClassificationUnknown {
		t.Fatalf("expected unknown, got %s", unknown.Status)
	}
}

func TestProbeSiteClassificationKnownInterfaces(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		body     string
		expected adminplusdomain.SupplierType
	}{
		{
			name: "new api status",
			path: "/api/status",
			body: `{
				"success": true,
				"message": "",
				"data": {
					"version": "v0.10.0",
					"quota_per_unit": 500000,
					"system_name": "New API",
					"setup": false,
					"register_enabled": true
				}
			}`,
			expected: adminplusdomain.SupplierTypeNewAPI,
		},
		{
			name: "sub2api public settings",
			path: "/api/v1/settings/public",
			body: `{
				"code": 0,
				"message": "success",
				"data": {
					"version": "0.11.3",
					"site_name": "Sub2API",
					"api_base_url": "https://api.example.com",
					"registration_enabled": true,
					"table_default_page_size": 20
				}
			}`,
			expected: adminplusdomain.SupplierTypeSub2API,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			service := NewService(nil, nil, nil, nil, server.Client())
			result := service.probeSiteClassification(context.Background(), &adminplusdomain.SiteDiscoveryItem{
				APIBaseURL: server.URL,
			})
			if result.Status != adminplusdomain.SiteDiscoveryClassificationSupported {
				t.Fatalf("expected supported, got %s", result.Status)
			}
			if result.ProviderType != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result.ProviderType)
			}
			if result.Confidence < 0.95 {
				t.Fatalf("expected high confidence, got %.2f", result.Confidence)
			}
		})
	}
}

func TestGenerateRegistrationPassword(t *testing.T) {
	password, err := generateRegistrationPassword(adminplusdomain.SupplierTypeNewAPI)
	if err != nil {
		t.Fatalf("generate password: %v", err)
	}
	if len(password) != defaultPasswordLength {
		t.Fatalf("expected length %d, got %d", defaultPasswordLength, len(password))
	}
	for _, chars := range []string{
		"abcdefghijkmnopqrstuvwxyz",
		"ABCDEFGHJKLMNPQRSTUVWXYZ",
		"23456789",
		"!@#_-",
	} {
		if !strings.ContainsAny(password, chars) {
			t.Fatalf("password %q does not contain a char from %q", password, chars)
		}
	}
}
