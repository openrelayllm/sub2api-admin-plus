package purity

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"net/http"
	"time"
)

func newPurityHTTPClient(allowPrivate bool) *http.Client {
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:               defaultHTTPTimeout,
		ResponseHeaderTimeout: defaultTokenAuditRoundTimeout + 5*time.Second,
		ValidateResolvedIP:    true,
		AllowPrivateHosts:     allowPrivate,
		MaxConnsPerHost:       2,
	})
	if err != nil {
		return &http.Client{Timeout: defaultHTTPTimeout}
	}
	return client
}

func (s *Service) clientForRun(options checkRunOptions) *http.Client {
	if s == nil {
		return &http.Client{Timeout: defaultHTTPTimeout}
	}
	if options.AllowPrivateHosts && s.accountHTTPClient != nil {
		return s.accountHTTPClient
	}
	if s.httpClient != nil {
		return s.httpClient
	}
	return &http.Client{Timeout: defaultHTTPTimeout}
}
