package runner

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HTTPRunner struct {
	client *http.Client
}

func NewHTTPRunner() *HTTPRunner {
	return &HTTPRunner{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
					MinVersion:         tls.VersionTLS12,
				},
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

func (r *HTTPRunner) Execute(ctx context.Context, target string, options map[string]interface{}) (map[string]interface{}, error) {
	fullURL, err := r.normalizeURL(target)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	method := getStringOption(options, "method", "GET")
	headers := getHeadersOption(options)
	followRedirects := getBoolOption(options, "follow_redirects", true)
	verifySSL := getBoolOption(options, "verify_ssl", true)

	client := r.configureClient(followRedirects, verifySSL)

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "NetScan-Agent/1.0")
	}

	start := time.Now()
	resp, err := client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	result := r.collectBasicInfo(resp, responseTime, fullURL)

	if resp.TLS != nil {
		result["ssl"] = r.collectSSLInfo(resp.TLS)
		result["protocol"] = resp.TLS.NegotiatedProtocol
		result["cipher_suite"] = resp.TLS.CipherSuite
	}

	if resp.Request.URL.String() != fullURL {
		result["final_url"] = resp.Request.URL.String()
		result["redirected"] = true
	}

	bodyInfo, err := r.readResponseBody(resp)
	if err != nil {
		result["body_error"] = err.Error()
	} else {
		result["body_preview"] = bodyInfo.preview
		result["content_length"] = bodyInfo.length
		result["content_type"] = resp.Header.Get("Content-Type")
	}

	return result, nil
}

func (r *HTTPRunner) normalizeURL(target string) (string, error) {
	if _, err := url.ParseRequestURI(target); err != nil {
		if httpURL, err := url.Parse("http://" + target); err == nil {
			return httpURL.String(), nil
		}
		return "", fmt.Errorf("invalid URL format: %s", target)
	}
	return target, nil
}

func (r *HTTPRunner) configureClient(followRedirects, verifySSL bool) *http.Client {
	transport := r.client.Transport.(*http.Transport).Clone()

	transport.TLSClientConfig.InsecureSkipVerify = !verifySSL

	client := *r.client
	client.Transport = transport

	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &client
}

func (r *HTTPRunner) collectBasicInfo(resp *http.Response, responseTime time.Duration, originalURL string) map[string]interface{} {
	headerMap := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headerMap[key] = values[0]
		}
	}

	return map[string]interface{}{
		"status_code":    resp.StatusCode,
		"status":         resp.Status,
		"headers":        headerMap,
		"response_time":  responseTime.Milliseconds(),
		"url":            originalURL,
		"proto":          resp.Proto,
		"content_length": resp.ContentLength,
	}
}

func (r *HTTPRunner) collectSSLInfo(tlsState *tls.ConnectionState) map[string]interface{} {
	if tlsState == nil || len(tlsState.PeerCertificates) == 0 {
		return map[string]interface{}{}
	}

	cert := tlsState.PeerCertificates[0]
	sslInfo := map[string]interface{}{
		"valid":                time.Now().Before(cert.NotAfter),
		"expires_at":           cert.NotAfter.Format(time.RFC3339),
		"issued_at":            cert.NotBefore.Format(time.RFC3339),
		"issuer":               cert.Issuer.String(),
		"subject":              cert.Subject.String(),
		"dns_names":            cert.DNSNames,
		"signature_algorithm":  cert.SignatureAlgorithm.String(),
		"public_key_algorithm": cert.PublicKeyAlgorithm.String(),
		"version":              cert.Version,
	}

	if len(tlsState.VerifiedChains) > 0 {
		sslInfo["chain_valid"] = true
		sslInfo["chain_length"] = len(tlsState.VerifiedChains[0])
	}

	return sslInfo
}

type bodyReadResult struct {
	preview string
	length  int64
}

func (r *HTTPRunner) readResponseBody(resp *http.Response) (*bodyReadResult, error) {
	if resp.Body == nil {
		return &bodyReadResult{}, nil
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, err
	}

	return &bodyReadResult{
		preview: string(bodyBytes),
		length:  int64(len(bodyBytes)),
	}, nil
}
