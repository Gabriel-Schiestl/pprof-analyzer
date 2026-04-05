package collector

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gabri/pprof-analyzer/internal/domain"
)

const (
	maxConcurrentProfiles = 3
	defaultMaxRetries     = 3
	defaultRetryDelay     = 5 * time.Second
	defaultCPUSeconds     = 30
)

// HTTPCollector implements ProfileCollector by fetching pprof endpoints over HTTP.
type HTTPCollector struct {
	httpClient        *http.Client
	maxRetries        int
	retryDelay        time.Duration
	cpuSampleDuration time.Duration
}

// NewHTTPCollector creates a collector with the given configuration.
func NewHTTPCollector(timeout time.Duration, maxRetries int, retryDelay time.Duration, cpuSampleDuration time.Duration) *HTTPCollector {
	return &HTTPCollector{
		httpClient:        &http.Client{Timeout: timeout},
		maxRetries:        maxRetries,
		retryDelay:        retryDelay,
		cpuSampleDuration: cpuSampleDuration,
	}
}

// Collect fetches all non-CPU profiles concurrently, then fetches the CPU profile.
func (c *HTTPCollector) Collect(ctx context.Context, endpoint domain.Endpoint) ([]domain.ProfileData, error) {
	// Separate CPU from other profiles — it has a long sampling window
	var regularTypes []domain.ProfileType
	for _, pt := range domain.AllProfileTypes {
		if pt != domain.ProfileCPU {
			regularTypes = append(regularTypes, pt)
		}
	}

	results := make([]domain.ProfileData, 0, len(domain.AllProfileTypes))
	var mu sync.Mutex

	sem := make(chan struct{}, maxConcurrentProfiles)
	var wg sync.WaitGroup
	var collectionErr error

	for _, pt := range regularTypes {
		wg.Add(1)
		pt := pt
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			data, err := c.fetchWithRetry(ctx, endpoint, pt, "")
			if err != nil {
				slog.Warn("profile collection failed", "endpoint", endpoint.Name, "profile", pt, "err", err)
				mu.Lock()
				if !isDomainError(err, domain.ErrProfileNotAvailable) {
					collectionErr = err
				}
				mu.Unlock()
				return
			}

			summary, _ := ParseToTextSummary(data, pt)
			mu.Lock()
			results = append(results, domain.ProfileData{
				Type:        pt,
				TextSummary: summary,
				CollectedAt: time.Now(),
				SizeBytes:   int64(len(data)),
			})
			mu.Unlock()
		}()
	}
	wg.Wait()

	// CPU profile last — requires sampling window
	cpuSeconds := int(c.cpuSampleDuration.Seconds())
	if cpuSeconds == 0 {
		cpuSeconds = defaultCPUSeconds
	}
	cpuQuery := fmt.Sprintf("?seconds=%d", cpuSeconds)
	cpuData, err := c.fetchWithRetry(ctx, endpoint, domain.ProfileCPU, cpuQuery)
	if err != nil {
		slog.Warn("CPU profile collection failed", "endpoint", endpoint.Name, "err", err)
	} else {
		summary, _ := ParseToTextSummary(cpuData, domain.ProfileCPU)
		mu.Lock()
		results = append(results, domain.ProfileData{
			Type:        domain.ProfileCPU,
			TextSummary: summary,
			CollectedAt: time.Now(),
			SizeBytes:   int64(len(cpuData)),
		})
		mu.Unlock()
	}

	if len(results) == 0 {
		if collectionErr != nil {
			return nil, collectionErr
		}
		return nil, &domain.RetryExhaustedError{
			Endpoint: endpoint.BaseURL,
			Attempts: c.maxRetries,
			Last:     domain.ErrEndpointUnreachable,
		}
	}

	return results, nil
}

func (c *HTTPCollector) fetchWithRetry(ctx context.Context, endpoint domain.Endpoint, profileType domain.ProfileType, query string) ([]byte, error) {
	url := fmt.Sprintf("%s/debug/pprof/%s%s", endpoint.BaseURL, string(profileType), query)

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		data, err := c.fetch(ctx, endpoint, url)
		if err == nil {
			return data, nil
		}

		lastErr = err

		// 404 means profile not available on this endpoint — don't retry
		if isDomainError(err, domain.ErrProfileNotAvailable) {
			return nil, err
		}

		if attempt < c.maxRetries-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * c.retryDelay
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return nil, &domain.RetryExhaustedError{
		Endpoint: endpoint.BaseURL,
		Attempts: c.maxRetries,
		Last:     lastErr,
	}
}

func (c *HTTPCollector) fetch(ctx context.Context, endpoint domain.Endpoint, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	applyAuth(req, endpoint.Credentials)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrEndpointUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrProfileNotAvailable
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", domain.ErrEndpointUnreachable, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}

func applyAuth(req *http.Request, creds domain.Credentials) {
	switch creds.AuthType {
	case domain.AuthBasic:
		req.SetBasicAuth(creds.Username, creds.Password)
	case domain.AuthBearerToken:
		req.Header.Set("Authorization", "Bearer "+creds.Token)
	}
}

func isDomainError(err error, target error) bool {
	if err == nil {
		return false
	}
	return err == target
}
