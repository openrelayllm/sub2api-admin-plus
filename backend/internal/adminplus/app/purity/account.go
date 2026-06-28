package purity

import (
	"context"
	"fmt"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	coreservice "github.com/Wei-Shaw/sub2api/internal/service"
	"strings"
)

func (s *Service) publicInputFromAccount(ctx context.Context, in AccountCheckInput) (PublicCheckInput, error) {
	if s == nil {
		return PublicCheckInput{}, infraerrors.InternalServer("PURITY_SERVICE_NOT_CONFIGURED", "purity service is not configured")
	}
	if s.accountResolver == nil {
		return PublicCheckInput{}, infraerrors.InternalServer("PURITY_ACCOUNT_RESOLVER_NOT_CONFIGURED", "account resolver is not configured")
	}
	if in.AccountID <= 0 {
		return PublicCheckInput{}, infraerrors.BadRequest("PURITY_ACCOUNT_ID_INVALID", "invalid account id")
	}
	requestedProvider := strings.TrimSpace(in.Provider)
	provider := normalizeProvider(requestedProvider)
	account, err := s.accountResolver.GetByID(ctx, in.AccountID)
	if err != nil {
		return PublicCheckInput{}, err
	}
	if account == nil {
		return PublicCheckInput{}, infraerrors.BadRequest("PURITY_ACCOUNT_UNSUPPORTED", "unsupported account")
	}
	if requestedProvider == "" {
		provider = normalizeProvider(account.Platform)
	}
	if provider == ProviderOpenAI && !account.IsOpenAIApiKey() {
		return PublicCheckInput{}, infraerrors.BadRequest("PURITY_ACCOUNT_UNSUPPORTED", "only OpenAI API key accounts can run purity checks")
	}
	if provider == ProviderAnthropic && (account.Platform != coreservice.PlatformAnthropic || account.Type != coreservice.AccountTypeAPIKey) {
		return PublicCheckInput{}, infraerrors.BadRequest("PURITY_ACCOUNT_UNSUPPORTED", "only Claude API key accounts can run purity checks")
	}
	if provider == ProviderGemini && (account.Platform != coreservice.PlatformGemini || account.Type != coreservice.AccountTypeAPIKey) {
		return PublicCheckInput{}, infraerrors.BadRequest("PURITY_ACCOUNT_UNSUPPORTED", "only Gemini API key accounts can run purity checks")
	}
	if provider != ProviderOpenAI && provider != ProviderAnthropic && provider != ProviderGemini {
		return PublicCheckInput{}, infraerrors.BadRequest("PURITY_PROVIDER_UNSUPPORTED", "only OpenAI, Claude and Gemini API key purity checks are supported")
	}
	apiKey := account.GetOpenAIApiKey()
	baseURL := account.GetOpenAIBaseURL()
	if provider == ProviderAnthropic {
		apiKey = account.GetCredential("api_key")
		baseURL = account.GetBaseURL()
	}
	if provider == ProviderGemini {
		apiKey = account.GetCredential("api_key")
		baseURL = account.GetGeminiBaseURL(defaultGeminiBaseURL)
	}
	if strings.TrimSpace(apiKey) == "" {
		return PublicCheckInput{}, infraerrors.BadRequest("PURITY_ACCOUNT_API_KEY_MISSING", "account api key is missing")
	}
	return PublicCheckInput{
		Provider:   provider,
		APIBaseURL: baseURL,
		APIKey:     apiKey,
		ModelID:    in.ModelID,
		ClientIP:   fmt.Sprintf("account:%d", in.AccountID),
	}, nil
}

func normalizeProvider(provider string) string {
	value := strings.ToLower(strings.TrimSpace(provider))
	alias := strings.NewReplacer("-", "_", " ", "_").Replace(value)
	switch alias {
	case "":
		return ProviderOpenAI
	case ProviderOpenAI, "openai_compatible", "openai_compat":
		return ProviderOpenAI
	case ProviderAnthropic, "claude", "anthropic_compatible", "claude_compatible", "claude_compat":
		return ProviderAnthropic
	case ProviderGemini, "google", "ai_studio", "google_ai", "google_ai_studio", "gemini_compatible", "gemini_compat":
		return ProviderGemini
	default:
		return value
	}
}

func normalizeAccessMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case AccessModeDeveloperAPI:
		return AccessModeDeveloperAPI
	case AccessModeAccount:
		return AccessModeAccount
	default:
		return AccessModeWeb
	}
}

func normalizeBillingMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case BillingModeAPIKeyMetered:
		return BillingModeAPIKeyMetered
	case BillingModeAccountInternal:
		return BillingModeAccountInternal
	default:
		return BillingModeCaptchaRateLimit
	}
}
