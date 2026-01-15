package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// HTTPError carries status/body for non-2xx responses.
// It lets callers decide if/when to retry.
type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("http error: %s %s status=%d body=%s", e.Method, e.URL, e.StatusCode, snippet(e.Body, 900))
}

func snippet(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// RetryConfig controls retry behavior.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration

	// If true, retry any 5xx.
	Retry5xx bool

	// Extra statuses to retry (e.g. 429, 408).
	RetryStatuses map[int]bool
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 8,
		BaseDelay:   700 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Retry5xx:    true,
		RetryStatuses: map[int]bool{
			http.StatusTooManyRequests:    true, // 429
			http.StatusRequestTimeout:     true, // 408
			http.StatusTooEarly:           true, // 425 (rare)
			http.StatusServiceUnavailable: true, // 503
			http.StatusBadGateway:         true, // 502
			http.StatusGatewayTimeout:     true, // 504
		},
	}
}

// DoWithRetry executes a request (built by buildReq) with retries.
// It always reads the full body (even on error) so the underlying TCP connection
// can be reused by http.Transport.
func DoWithRetry(
	ctx context.Context,
	client *http.Client,
	buildReq func(context.Context) (*http.Request, error),
	cfg RetryConfig,
) (*http.Response, []byte, error) {
	if cfg.MaxAttempts <= 0 {
		cfg = DefaultRetryConfig()
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 700 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.RetryStatuses == nil {
		cfg.RetryStatuses = DefaultRetryConfig().RetryStatuses
	}

	var lastErr error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		req, err := buildReq(ctx)
		if err != nil {
			return nil, nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			if isRetryableNetErr(err) {
				lastErr = err
				if attempt < cfg.MaxAttempts {
					if err := sleepBackoff(ctx, attempt, cfg.BaseDelay, cfg.MaxDelay, 0); err != nil {
						return nil, nil, err
					}
					continue
				}
			}
			return nil, nil, err
		}

		body, readErr := readAndClose(resp.Body)
		if readErr != nil {
			if isRetryableNetErr(readErr) {
				lastErr = readErr
				if attempt < cfg.MaxAttempts {
					if err := sleepBackoff(ctx, attempt, cfg.BaseDelay, cfg.MaxDelay, 0); err != nil {
						return nil, nil, err
					}
					continue
				}
			}
			return resp, body, readErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, body, nil
		}

		herr := &HTTPError{
			Method:     req.Method,
			URL:        req.URL.String(),
			StatusCode: resp.StatusCode,
			Header:     resp.Header.Clone(),
			Body:       body,
		}

		retryAfter := ParseRetryAfter(resp)
		if isRetryableStatus(resp.StatusCode, cfg) {
			lastErr = herr
			if attempt < cfg.MaxAttempts {
				if err := sleepBackoff(ctx, attempt, cfg.BaseDelay, cfg.MaxDelay, retryAfter); err != nil {
					return nil, nil, err
				}
				continue
			}
		}

		return resp, body, herr
	}

	if lastErr != nil {
		return nil, nil, lastErr
	}
	return nil, nil, errors.New("httpx: request failed")
}

func readAndClose(rc io.ReadCloser) ([]byte, error) {
	defer rc.Close()
	return io.ReadAll(rc)
}

func isRetryableStatus(code int, cfg RetryConfig) bool {
	if cfg.RetryStatuses != nil && cfg.RetryStatuses[code] {
		return true
	}
	if cfg.Retry5xx && code >= 500 && code <= 599 {
		return true
	}
	return false
}

func sleepBackoff(ctx context.Context, attempt int, base, max time.Duration, retryAfter time.Duration) error {
	sleep := retryAfter
	if sleep <= 0 {
		sleep = base * time.Duration(1<<(attempt-1))
		if sleep > max {
			sleep = max
		}
		// jitter 0..400ms
		sleep += time.Duration(rand.Intn(400)) * time.Millisecond
	}

	t := time.NewTimer(sleep)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func isRetryableNetErr(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var nerr net.Error
	if errors.As(err, &nerr) {
		return nerr.Timeout() || nerr.Temporary()
	}

	// common transient I/O errors
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection reset") || strings.Contains(msg, "broken pipe") || strings.Contains(msg, "eof") {
		return true
	}
	return false
}

// ParseRetryAfter parses Retry-After header (seconds or HTTP date).
// Returns 0 when header is missing/invalid.
func ParseRetryAfter(resp *http.Response) time.Duration {
	v := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

// DoJSON is a convenience wrapper over DoWithRetry that unmarshals JSON.
func DoJSON(
	ctx context.Context,
	client *http.Client,
	buildReq func(context.Context) (*http.Request, error),
	out any,
	cfg RetryConfig,
) error {
	_, body, err := DoWithRetry(ctx, client, buildReq, cfg)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("json parse error: %w body=%s", err, snippet(body, 900))
	}
	return nil
}
