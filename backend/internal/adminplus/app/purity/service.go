package purity

import (
	"context"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	defaultCheckTimeout    = 120 * time.Second
	defaultHTTPTimeout     = 45 * time.Second
	tokenAuditTimeout      = 70 * time.Second
	tokenAuditRoundTimeout = 10 * time.Second
	maxProbeBodyBytes      = 256 * 1024
	maxFingerprintBytes    = 4 * 1024
	maxErrorMessageLen     = 500
	maxAPIKeyLength        = 8192
	tokenAuditSamples      = 11
	probePNGBase64         = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAAJ0lEQVR42mN44WOLFRm0/MeKGEY10ESD0tE4rOjXGVGsaFQDTTQAAIwskRBmlXeKAAAAAElFTkSuQmCC"
	probePNGData           = "data:image/png;base64," + probePNGBase64

	errorClassAccountBalanceInsufficient = "account_balance_insufficient"
)

type Service struct {
	repo              Repository
	accountResolver   AccountResolver
	httpClient        *http.Client
	accountHTTPClient *http.Client
	now               func() time.Time
	limiter           *publicLimiter
	allowPrivateHosts bool
}

func NewService(repo Repository) *Service {
	client := newPurityHTTPClient(false)
	accountClient := newPurityHTTPClient(true)
	return &Service{
		repo:              repo,
		httpClient:        client,
		accountHTTPClient: accountClient,
		now:               time.Now,
		limiter:           newPublicLimiter(),
	}
}

func NewServiceWithAccountResolver(repo Repository, accountResolver AccountResolver) *Service {
	service := NewService(repo)
	service.accountResolver = accountResolver
	return service
}

func (s *Service) RunPublicCheck(ctx context.Context, in PublicCheckInput) (*PublicReport, error) {
	return s.runCheck(ctx, in, nil, checkRunOptions{
		EnforceRateLimit:  true,
		AllowPrivateHosts: s != nil && s.allowPrivateHosts,
		AccessMode:        AccessModeWeb,
		BillingMode:       BillingModeCaptchaRateLimit,
	})
}

func (s *Service) RunPublicCheckStream(ctx context.Context, in PublicCheckInput, emit PublicCheckEventSink) (*PublicReport, error) {
	return s.runCheck(ctx, in, emit, checkRunOptions{
		EnforceRateLimit:  true,
		AllowPrivateHosts: s != nil && s.allowPrivateHosts,
		AccessMode:        AccessModeWeb,
		BillingMode:       BillingModeCaptchaRateLimit,
	})
}

func (s *Service) RunDeveloperAPICheck(ctx context.Context, in PublicCheckInput) (*PublicReport, error) {
	return s.runCheck(ctx, in, nil, checkRunOptions{
		EnforceRateLimit:  false,
		AllowPrivateHosts: s != nil && s.allowPrivateHosts,
		AccessMode:        AccessModeDeveloperAPI,
		BillingMode:       BillingModeAPIKeyMetered,
	})
}

func (s *Service) RunDeveloperAPICheckStream(ctx context.Context, in PublicCheckInput, emit PublicCheckEventSink) (*PublicReport, error) {
	return s.runCheck(ctx, in, emit, checkRunOptions{
		EnforceRateLimit:  false,
		AllowPrivateHosts: s != nil && s.allowPrivateHosts,
		AccessMode:        AccessModeDeveloperAPI,
		BillingMode:       BillingModeAPIKeyMetered,
	})
}

func (s *Service) RunAccountCheck(ctx context.Context, in AccountCheckInput) (*PublicReport, error) {
	return s.RunAccountCheckStream(ctx, in, nil)
}

func (s *Service) RunAccountCheckStream(ctx context.Context, in AccountCheckInput, emit PublicCheckEventSink) (*PublicReport, error) {
	publicInput, err := s.publicInputFromAccount(ctx, in)
	if err != nil {
		return nil, err
	}
	return s.runCheck(ctx, publicInput, emit, checkRunOptions{
		EnforceRateLimit:  false,
		AllowPrivateHosts: true,
		AccessMode:        AccessModeAccount,
		BillingMode:       BillingModeAccountInternal,
	})
}

type checkRunOptions struct {
	EnforceRateLimit  bool
	AllowPrivateHosts bool
	AccessMode        string
	BillingMode       string
}

func (s *Service) runCheck(ctx context.Context, in PublicCheckInput, emit PublicCheckEventSink, options checkRunOptions) (*PublicReport, error) {
	if s == nil {
		return nil, infraerrors.InternalServer("PURITY_SERVICE_NOT_CONFIGURED", "purity service is not configured")
	}
	requestedProvider := strings.TrimSpace(in.Provider)
	provider := normalizeProvider(requestedProvider)
	if provider == ProviderAnthropic {
		return s.runClaudeCheck(ctx, in, emit, options)
	}
	if provider == ProviderGemini {
		return s.runGeminiCheck(ctx, in, emit, options)
	}
	if provider != ProviderOpenAI {
		return nil, infraerrors.BadRequest("PURITY_PROVIDER_UNSUPPORTED", "only OpenAI, Claude and Gemini API key purity checks are supported")
	}
	return s.runOpenAICheck(ctx, in, emit, options)
}
