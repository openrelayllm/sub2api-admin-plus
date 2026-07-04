package kanban

import (
	"bytes"
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	adminplusdomain "github.com/Wei-Shaw/sub2api/internal/adminplus/domain"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	sharedhttp "github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"golang.org/x/net/html"
)

const (
	defaultMarketPriceFetchTimeout = 12 * time.Second
	maxMarketPricePageBytes        = 512 * 1024
	maxMarketPriceTextLength       = 160 * 1024
)

type MarketPriceImportURLInput struct {
	SourceType      string
	SourceName      string
	SourceURL       string
	SiteID          int64
	SupplierID      int64
	DefaultCurrency string
	Confidence      float64
	ObservedAt      *time.Time
}

type MarketPriceImportURLResult struct {
	Items       []*adminplusdomain.MarketPriceSnapshot `json:"items"`
	Total       int                                    `json:"total"`
	SourceURL   string                                 `json:"source_url"`
	ContentType string                                 `json:"content_type,omitempty"`
	TextLength  int                                    `json:"text_length"`
}

func (s *Service) ImportMarketPricesFromURL(ctx context.Context, in MarketPriceImportURLInput) (*MarketPriceImportURLResult, error) {
	if s == nil || s.repo == nil {
		return nil, internalError("kanban service is not configured")
	}
	sourceURL, err := normalizeMarketPriceFetchURL(in.SourceURL)
	if err != nil {
		return nil, err
	}
	body, contentType, err := s.fetchMarketPricePage(ctx, sourceURL)
	if err != nil {
		return nil, err
	}
	text, err := marketPriceTextFromPage(body, contentType)
	if err != nil {
		return nil, err
	}
	parseResult, err := s.ParseMarketPrices(ctx, MarketPriceParseInput{
		SourceType:      in.SourceType,
		SourceName:      in.SourceName,
		SourceURL:       sourceURL,
		SiteID:          in.SiteID,
		SupplierID:      in.SupplierID,
		DefaultCurrency: in.DefaultCurrency,
		Confidence:      in.Confidence,
		Text:            text,
		ObservedAt:      in.ObservedAt,
	})
	if err != nil {
		return nil, err
	}
	return &MarketPriceImportURLResult{
		Items:       parseResult.Items,
		Total:       parseResult.Total,
		SourceURL:   sourceURL,
		ContentType: contentType,
		TextLength:  len([]rune(text)),
	}, nil
}

func normalizeMarketPriceFetchURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", badRequest("KANBAN_PRICE_SOURCE_URL_REQUIRED", "source_url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", badRequest("KANBAN_PRICE_SOURCE_URL_INVALID", "invalid source url")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", badRequest("KANBAN_PRICE_SOURCE_URL_UNSUPPORTED", "only http and https price source urls are supported")
	}
	if parsed.User != nil {
		return "", badRequest("KANBAN_PRICE_SOURCE_URL_CREDENTIALS_UNSUPPORTED", "source url must not contain credentials")
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func (s *Service) fetchMarketPricePage(ctx context.Context, sourceURL string) ([]byte, string, error) {
	client, err := s.marketPriceHTTPClient()
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Sub2API-Admin-Plus-Kanban/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,application/json;q=0.8,*/*;q=0.2")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", infraerrors.New(resp.StatusCode, "KANBAN_PRICE_PAGE_FETCH_FAILED", "failed to fetch price source page")
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if !marketPriceContentTypeSupported(contentType) {
		return nil, contentType, infraerrors.New(http.StatusUnsupportedMediaType, "KANBAN_PRICE_PAGE_CONTENT_TYPE_UNSUPPORTED", "price source page content type is not supported")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMarketPricePageBytes+1))
	if err != nil {
		return nil, contentType, err
	}
	if len(body) > maxMarketPricePageBytes {
		return nil, contentType, infraerrors.New(http.StatusRequestEntityTooLarge, "KANBAN_PRICE_PAGE_TOO_LARGE", "price source page is too large")
	}
	return body, contentType, nil
}

func (s *Service) marketPriceHTTPClient() (*http.Client, error) {
	if s != nil && s.marketPriceClient != nil {
		return s.marketPriceClient, nil
	}
	client, err := sharedhttp.GetClient(sharedhttp.Options{
		Timeout:               defaultMarketPriceFetchTimeout,
		ResponseHeaderTimeout: 6 * time.Second,
		ValidateResolvedIP:    true,
		MaxConnsPerHost:       2,
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func marketPriceContentTypeSupported(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}
	if mediaType == "" {
		return true
	}
	return mediaType == "text/html" ||
		mediaType == "application/xhtml+xml" ||
		mediaType == "text/plain" ||
		mediaType == "application/json" ||
		strings.HasSuffix(mediaType, "+json")
}

func marketPriceTextFromPage(body []byte, contentType string) (string, error) {
	if len(body) == 0 {
		return "", badRequest("KANBAN_PRICE_PAGE_EMPTY", "price source page is empty")
	}
	mediaType, _, _ := mime.ParseMediaType(contentType)
	if mediaType == "text/html" || mediaType == "application/xhtml+xml" || looksLikeHTML(body) {
		text := extractHTMLText(body)
		if strings.TrimSpace(text) == "" {
			return "", badRequest("KANBAN_PRICE_PAGE_TEXT_EMPTY", "price source page has no readable text")
		}
		return trimMarketPriceText(text), nil
	}
	return trimMarketPriceText(string(body)), nil
}

func looksLikeHTML(body []byte) bool {
	trimmed := strings.ToLower(strings.TrimSpace(string(body[:minInt(len(body), 256)])))
	return strings.HasPrefix(trimmed, "<!doctype html") || strings.HasPrefix(trimmed, "<html") || strings.Contains(trimmed, "<body")
}

func extractHTMLText(body []byte) string {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return string(body)
	}
	var parts []string
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.ElementNode && marketPriceHTMLNodeSkipped(node.Data) {
			return
		}
		if node.Type == html.TextNode {
			if text := strings.TrimSpace(node.Data); text != "" {
				parts = append(parts, text)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if node.Type == html.ElementNode && marketPriceHTMLNodeBreaksLine(node.Data) {
			parts = append(parts, "\n")
		}
	}
	walk(root)
	return strings.Join(parts, " ")
}

func marketPriceHTMLNodeSkipped(name string) bool {
	switch strings.ToLower(name) {
	case "script", "style", "noscript", "svg", "canvas", "iframe", "template":
		return true
	default:
		return false
	}
}

func marketPriceHTMLNodeBreaksLine(name string) bool {
	switch strings.ToLower(name) {
	case "br", "p", "div", "li", "tr", "section", "article", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func trimMarketPriceText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" {
			out = append(out, line)
		}
	}
	joined := strings.TrimSpace(strings.Join(out, "\n"))
	if len([]rune(joined)) <= maxMarketPriceTextLength {
		return joined
	}
	runes := []rune(joined)
	return string(runes[:maxMarketPriceTextLength])
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
