package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	"github.com/Wei-Shaw/sub2api/internal/adminplus/ports"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestSessionProfileClientProbeSub2APIUserProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer browser-access-token", r.Header.Get("Authorization"))
		require.Equal(t, "sid=abc; theme=dark", r.Header.Get("Cookie"))
		require.Equal(t, "https://relay.example.com", r.Header.Get("Origin"))
		require.Equal(t, "https://relay.example.com/dashboard", r.Header.Get("Referer"))
		require.Equal(t, "csrf-token", r.Header.Get("X-CSRF-Token"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/user/profile":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{
				"data": {
					"id": 42,
					"email": "ops@example.com",
					"username": "ops",
					"role": "user",
					"status": "enabled",
					"balance": 12.34,
					"concurrency": 8,
					"allowed_groups": [1, 2]
				}
			}`))
		case "/api/v1/groups/available":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"data":[{"id":1,"name":"GPT"}]}`))
		case "/api/v1/channels/available":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"data":[{"name":"OpenAI"}]}`))
		case "/api/v1/announcements":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"data":[{"id":1,"title":"公告","content":"通知"}]}`))
		case "/api/v1/usage":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "1", r.URL.Query().Get("page"))
			require.Equal(t, "1", r.URL.Query().Get("page_size"))
			_, _ = w.Write([]byte(`{"data":[]}`))
		case "/api/v1/keys":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "1", r.URL.Query().Get("page"))
			require.Equal(t, "1", r.URL.Query().Get("page_size"))
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	result, err := client.ProbeSub2APIUserProfile(context.Background(), ports.SessionProbeInput{
		SupplierID: 7,
		Origin:     "https://relay.example.com",
		APIBaseURL: server.URL,
		Bundle: map[string]any{
			"access_token": "browser-access-token",
			"csrf_token":   "csrf-token",
			"required_headers": map[string]any{
				"cookie":  "sid=abc; theme=dark",
				"origin":  "https://relay.example.com",
				"referer": "https://relay.example.com/dashboard",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), result.SupplierID)
	require.Equal(t, "valid", result.Status)
	require.Equal(t, "sub2api", result.SystemType)
	require.Equal(t, server.URL, result.APIBaseURL)
	require.NotNil(t, result.BalanceCents)
	require.Equal(t, int64(1234), *result.BalanceCents)
	require.Equal(t, "USD", result.BalanceCurrency)
	require.True(t, result.Capabilities["can_read_profile"])
	require.True(t, result.Capabilities["can_read_balance"])
	require.True(t, result.Capabilities["can_read_groups"])
	require.True(t, result.Capabilities["can_read_rates"])
	require.True(t, result.Capabilities["can_read_announcements"])
	require.True(t, result.Capabilities["can_create_key"])
	require.True(t, result.Capabilities["can_read_billing"])
	require.Equal(t, int64(42), result.Profile.ID)
	require.Equal(t, "ops@example.com", result.Profile.Email)
	require.Equal(t, []int64{1, 2}, result.Profile.AllowedGroups)
}

func TestSessionProfileClientDirectLogin(t *testing.T) {
	var sawSettings bool
	var sawLogin bool
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/settings/public":
			require.Equal(t, http.MethodGet, r.Method)
			sawSettings = true
			_, _ = w.Write([]byte(`{"data":{"login_agreement_revision":"rev-1"}}`))
		case "/api/v1/auth/login":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			require.Equal(t, serverURL, r.Header.Get("Origin"))
			require.Equal(t, serverURL+"/", r.Header.Get("Referer"))
			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Equal(t, "ops@example.com", payload["email"])
			require.Equal(t, "secret", payload["password"])
			require.Equal(t, "rev-1", payload["login_agreement_revision"])
			sawLogin = true
			_, _ = w.Write([]byte(`{"data":{"access_token":"direct-access-token","expires_in":3600}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewSessionProfileClient(server.Client())
	result, err := client.DirectLogin(context.Background(), ports.DirectLoginInput{
		SupplierID: 7,
		Origin:     server.URL,
		APIBaseURL: server.URL,
		Username:   "ops@example.com",
		Password:   "secret",
	})

	require.NoError(t, err)
	require.True(t, sawSettings)
	require.True(t, sawLogin)
	require.Equal(t, int64(7), result.SupplierID)
	require.Equal(t, server.URL, result.Origin)
	require.Equal(t, server.URL, result.APIBaseURL)
	require.Equal(t, "direct-access-token", result.SessionBundle["access_token"])
	require.Equal(t, "direct_login", result.SessionBundle["session_source"])
	tokens, ok := result.SessionBundle["tokens"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "direct-access-token", tokens["access_token"])
	contextValue, ok := result.SessionBundle["context"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "direct_login", contextValue["login_method"])
	require.NotNil(t, result.ExpiresAt)
}

func TestSessionProfileClientDirectLoginClassifiesCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/settings/public":
			_, _ = w.Write([]byte(`{"data":{}}`))
		case "/api/v1/auth/login":
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"message":"turnstile captcha required"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	_, err := client.DirectLogin(context.Background(), ports.DirectLoginInput{
		SupplierID: 7,
		Origin:     server.URL,
		APIBaseURL: server.URL,
		Username:   "ops@example.com",
		Password:   "secret",
	})

	require.Error(t, err)
	require.Equal(t, "LOGIN_CAPTCHA_REQUIRED", infraerrors.Reason(err))
}

func TestSessionProfileClientCreateKey(t *testing.T) {
	var listCalled bool
	var createCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer browser-access-token", r.Header.Get("Authorization"))
		require.Equal(t, "sid=abc", r.Header.Get("Cookie"))
		require.Equal(t, "/api/v1/keys", r.URL.Path)
		if r.Method == http.MethodGet {
			listCalled = true
			require.Equal(t, "1", r.URL.Query().Get("page"))
			require.Equal(t, "100", r.URL.Query().Get("page_size"))
			require.Equal(t, "ops-key", r.URL.Query().Get("search"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"items":[]}}`))
			return
		}

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		createCalled = true

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "ops-key", payload["name"])
		require.Equal(t, float64(10), payload["group_id"])
		require.Equal(t, float64(25), payload["quota"])
		require.Equal(t, float64(7), payload["expires_in_days"])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"id": 99,
				"name": "ops-key",
				"key": "sk-supplier-secret",
				"group_id": 10,
				"status": "active"
			}
		}`))
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	expires := 7
	result, err := client.CreateKey(context.Background(), ports.SessionProbeInput{
		SupplierID: 7,
		APIBaseURL: server.URL + "/api/v1",
		Bundle: map[string]any{
			"access_token": "browser-access-token",
			"required_headers": map[string]any{
				"cookie": "sid=abc",
			},
		},
	}, ports.CreateProviderKeyInput{
		SupplierID:      7,
		ExternalGroupID: "10",
		Name:            "ops-key",
		QuotaUSD:        25,
		ExpiresInDays:   &expires,
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), result.SupplierID)
	require.Equal(t, "99", result.ExternalKeyID)
	require.Equal(t, "10", result.ExternalGroupID)
	require.Equal(t, "ops-key", result.Name)
	require.Equal(t, "sk-supplier-secret", result.Secret)
	require.Equal(t, "active", result.Status)
	require.NotContains(t, result.RawPayload, "key")
	require.True(t, listCalled)
	require.True(t, createCalled)
}

func TestSessionProfileClientCreateKeyReusesExistingProviderKey(t *testing.T) {
	var createCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/keys", r.URL.Path)
		require.Equal(t, "Bearer browser-access-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{
				"data": {
					"items": [
						{
							"id": 99,
							"name": "ops-key",
							"key": "sk-existing-secret",
							"group_routes": [
								{"group_id": 10}
							],
							"status": "active"
						}
					]
				}
			}`))
		case http.MethodPost:
			createCalled = true
			http.Error(w, "must not create duplicate key", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	result, err := client.CreateKey(context.Background(), ports.SessionProbeInput{
		SupplierID: 7,
		APIBaseURL: server.URL,
		Bundle: map[string]any{
			"access_token": "browser-access-token",
		},
	}, ports.CreateProviderKeyInput{
		SupplierID:      7,
		ExternalGroupID: "10",
		Name:            "ops-key",
		QuotaUSD:        25,
	})

	require.NoError(t, err)
	require.False(t, createCalled)
	require.Equal(t, "99", result.ExternalKeyID)
	require.Equal(t, "10", result.ExternalGroupID)
	require.Equal(t, "ops-key", result.Name)
	require.Equal(t, "sk-existing-secret", result.Secret)
}

func TestSessionProfileClientReadGroups(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer browser-access-token", r.Header.Get("Authorization"))
		require.Equal(t, "sid=abc", r.Header.Get("Cookie"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/groups/available":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": 10,
						"name": "GPT-5.5 Low Cost",
						"description": "cheap upstream pool",
						"platform": "openai",
						"rate_multiplier": 0.8,
						"rpm_limit": 120,
						"is_exclusive": true,
						"status": "active",
						"daily_limit_usd": 100,
						"allow_image_generation": true
					},
					{
						"id": 11,
						"name": "Claude",
						"platform": "anthropic",
						"rate_multiplier": 1.2,
						"status": "disabled"
					}
				]
			}`))
		case "/api/v1/groups/rates":
			_, _ = w.Write([]byte(`{"data":{"10":0.7}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	result, err := client.ReadGroups(context.Background(), ports.SessionProbeInput{
		SupplierID: 7,
		Origin:     server.URL,
		APIBaseURL: server.URL + "/api/v1",
		Bundle: map[string]any{
			"access_token": "browser-access-token",
			"required_headers": map[string]any{
				"cookie": "sid=abc",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), result.SupplierID)
	require.Equal(t, "sub2api", result.SystemType)
	require.Len(t, result.Groups, 2)
	require.Equal(t, "10", result.Groups[0].ExternalGroupID)
	require.Equal(t, "openai", result.Groups[0].ProviderFamily)
	require.Equal(t, 0.8, result.Groups[0].RateMultiplier)
	require.NotNil(t, result.Groups[0].UserRateMultiplier)
	require.Equal(t, 0.7, *result.Groups[0].UserRateMultiplier)
	require.Equal(t, 0.7, result.Groups[0].EffectiveRateMultiplier)
	require.NotNil(t, result.Groups[0].RPMLimit)
	require.Equal(t, int64(120), *result.Groups[0].RPMLimit)
	require.True(t, result.Groups[0].IsPrivate)
	require.True(t, result.Groups[0].AllowImageGeneration)
	require.Equal(t, "disabled", result.Groups[1].Status)
}

func TestSessionProfileClientReadRates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer browser-access-token", r.Header.Get("Authorization"))
		require.Equal(t, "sid=abc", r.Header.Get("Cookie"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/rates/snapshots":
			_, _ = w.Write([]byte(`{"data":[]}`))
		case "/api/v1/channels/available":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"name": "OpenAI",
						"supported_models": [
							{
								"name": "gpt-5-mini",
								"platform": "openai",
								"pricing": {
									"billing_mode": "token",
									"input_price": 0.0000015,
									"output_price": 0.000006,
									"cache_read_price_micros": 250000
								}
							}
						]
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	result, err := client.ReadRates(context.Background(), ports.SessionProbeInput{
		SupplierID: 7,
		Origin:     server.URL,
		APIBaseURL: server.URL,
		Bundle: map[string]any{
			"access_token": "browser-access-token",
			"required_headers": map[string]any{
				"cookie": "sid=abc",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), result.SupplierID)
	require.Equal(t, "sub2api", result.SystemType)
	require.Len(t, result.Entries, 3)

	input := findRateEntry(t, result.Entries, "input")
	require.Equal(t, "gpt-5-mini", input.Model)
	require.Equal(t, "1m_tokens", input.Unit)
	require.Equal(t, int64(1500000), input.PriceMicros)
	require.Equal(t, int64(6000000), findRateEntry(t, result.Entries, "output").PriceMicros)
	require.Equal(t, int64(250000), findRateEntry(t, result.Entries, "cache_read").PriceMicros)
}

func TestSessionProfileClientReadRatesParsesNestedAvailableChannels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer browser-access-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/rates/snapshots":
			http.NotFound(w, r)
		case "/api/v1/channels/available":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"name": "CODEX",
						"platforms": [
							{
								"platform": "openai",
								"groups": [
									{"id": 59, "name": "CODEX", "rate_multiplier": 0.12}
								],
								"supported_models": [
									{
										"name": "codex-auto-review",
										"platform": "openai",
										"pricing": {
											"billing_mode": "token",
											"input_price": 0.000001,
											"output_price": 0.000004,
											"cache_write_price": 0.0000005,
											"cache_read_price": 0.0000001,
											"image_input_price": 0.000002,
											"image_cache_read_price": 0.0000002,
											"image_output_price": 0.000008,
											"per_request_price": 0.01
										}
									}
								]
							}
						]
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	result, err := client.ReadRates(context.Background(), ports.SessionProbeInput{
		SupplierID: 7,
		Origin:     server.URL,
		APIBaseURL: server.URL,
		Bundle: map[string]any{
			"access_token": "browser-access-token",
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), result.SupplierID)
	require.Len(t, result.Entries, 8)
	require.Equal(t, int64(1000000), findRateEntry(t, result.Entries, "input").PriceMicros)
	require.Equal(t, int64(4000000), findRateEntry(t, result.Entries, "output").PriceMicros)
	require.Equal(t, int64(500000), findRateEntry(t, result.Entries, "cache_write").PriceMicros)
	require.Equal(t, int64(100000), findRateEntry(t, result.Entries, "cache_read").PriceMicros)
	require.Equal(t, int64(2000000), findRateEntry(t, result.Entries, "image_input").PriceMicros)
	require.Equal(t, int64(200000), findRateEntry(t, result.Entries, "image_cache_read").PriceMicros)
	require.Equal(t, int64(8000000), findRateEntry(t, result.Entries, "image_output").PriceMicros)
	require.Equal(t, int64(10000), findRateEntry(t, result.Entries, "per_request").PriceMicros)
}

func TestSessionProfileClientReadAnnouncements(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer browser-access-token", r.Header.Get("Authorization"))
		require.Equal(t, "sid=abc", r.Header.Get("Cookie"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/user/profile":
			_, _ = w.Write([]byte(`{"data":{"id":42,"email":"ops@example.com","balance":12.34}}`))
		case "/api/v1/payment/checkout-info":
			_, _ = w.Write([]byte(`{
				"data": {
					"currency": "usd",
					"balance_recharge_multiplier": 1.2,
					"global_min": 100
				}
			}`))
		case "/api/v1/announcements":
			_, _ = w.Write([]byte(`{
				"data": [
					{
						"id": "notice-1",
						"title": "Limited offer",
						"content": "Weekend limited sale",
						"type": "limited_offer",
						"discount_percent": "15%"
					},
					{
						"id": "notice-2",
						"title": "维护通知",
						"content": "今晚模型网关维护，部分请求可能中断"
					}
				]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	result, err := client.ReadAnnouncements(context.Background(), ports.SessionProbeInput{
		SupplierID: 7,
		Origin:     server.URL,
		APIBaseURL: server.URL,
		Bundle: map[string]any{
			"access_token": "browser-access-token",
			"required_headers": map[string]any{
				"cookie": "sid=abc",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), result.SupplierID)
	require.Equal(t, "sub2api", result.SystemType)
	require.Len(t, result.Announcements, 3)
	require.Equal(t, "充值倍率公告 20.00%", result.Announcements[0].Title)
	require.Equal(t, int64(10000), result.Announcements[0].MinRechargeCents)
	require.Equal(t, int64(1234), result.Announcements[0].BalanceCents)
	require.NotNil(t, result.Announcements[0].BonusPercent)
	require.InEpsilon(t, 20.0, *result.Announcements[0].BonusPercent, 0.0001)
	require.Equal(t, "Limited offer", result.Announcements[1].Title)
	require.Equal(t, adminplusdomain.AnnouncementTypeLimitedOffer, result.Announcements[1].Type)
	require.Equal(t, "维护通知", result.Announcements[2].Title)
	require.Equal(t, adminplusdomain.AnnouncementTypeMaintenance, result.Announcements[2].Type)
	require.Equal(t, "maintenance", result.Announcements[2].RawPayload["classification"])
}

func TestSessionProfileClientReadBilling(t *testing.T) {
	startedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(24 * time.Hour)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer browser-access-token", r.Header.Get("Authorization"))
		require.Equal(t, "sid=abc", r.Header.Get("Cookie"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/usage":
			require.Equal(t, startedAt.Format(time.RFC3339), r.URL.Query().Get("started_at"))
			require.Equal(t, endedAt.Format(time.RFC3339), r.URL.Query().Get("ended_at"))
			require.Equal(t, startedAt.Format(time.RFC3339), r.URL.Query().Get("from"))
			require.Equal(t, endedAt.Format(time.RFC3339), r.URL.Query().Get("to"))
			_, _ = w.Write([]byte(`{
				"data": {
					"items": [
						{
							"id": 91,
							"request_id": "req-1",
							"api_key_name": "ops-key",
							"model": "gpt-5-mini",
							"endpoint": "/v1/responses",
							"request_type": "responses",
							"billing_mode": "token",
							"currency": "usd",
							"cost": 1.23,
							"input_tokens": 1000,
							"output_tokens": 500,
							"cache_read_tokens": 200,
							"first_token_ms": 680,
							"duration_ms": 2200,
							"user_agent": "OpenAI/Python",
							"started_at": "2026-06-20T10:00:00Z",
							"ended_at": "2026-06-20T10:00:02Z",
							"access_token": "must-not-persist",
							"headers": {
								"cookie": "sid=secret",
								"x-safe": "kept"
							}
						}
					]
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewSessionProfileClient(server.Client())
	result, err := client.ReadBilling(context.Background(), ports.SessionProbeInput{
		SupplierID: 7,
		Origin:     server.URL,
		APIBaseURL: server.URL,
		Bundle: map[string]any{
			"access_token": "browser-access-token",
			"required_headers": map[string]any{
				"cookie": "sid=abc",
			},
		},
	}, ports.ReadBillingInput{
		SupplierID: 7,
		StartedAt:  startedAt,
		EndedAt:    endedAt,
	})

	require.NoError(t, err)
	require.Equal(t, int64(7), result.SupplierID)
	require.Equal(t, "sub2api", result.SystemType)
	require.Len(t, result.Lines, 1)
	line := result.Lines[0]
	require.Equal(t, "91", line.ExternalBillID)
	require.Equal(t, "req-1", line.ExternalRequestID)
	require.Equal(t, "ops-key", line.APIKeyName)
	require.Equal(t, "gpt-5-mini", line.Model)
	require.Equal(t, int64(123), line.CostCents)
	require.Equal(t, int64(1000), line.InputTokens)
	require.Equal(t, int64(500), line.OutputTokens)
	require.Equal(t, int64(200), line.CacheReadTokens)
	require.Equal(t, int64(680), line.FirstTokenMS)
	require.Equal(t, "usd", line.Currency)
	require.NotContains(t, line.RawPayload, "access_token")
	headers, ok := line.RawPayload["headers"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, headers, "cookie")
	require.Equal(t, "kept", headers["x-safe"])
}

func findRateEntry(t *testing.T, entries []ports.ProviderRateEntry, priceItem string) ports.ProviderRateEntry {
	t.Helper()
	for _, entry := range entries {
		if entry.PriceItem == priceItem {
			return entry
		}
	}
	require.Failf(t, "rate entry not found", "price_item=%s", priceItem)
	return ports.ProviderRateEntry{}
}
