package broker

import (
	"math/rand/v2"
	"net/http"
	"time"
)

// retryConfig defines retry behaviour for HTTP requests.
type retryConfig struct {
	maxAttempts int           // total attempts (1 = no retry)
	baseDelay   time.Duration // initial delay before first retry
	maxDelay    time.Duration // cap on delay
}

// defaultRetry is used for all Supabase RPC calls.
var defaultRetry = retryConfig{
	maxAttempts: 3,
	baseDelay:   200 * time.Millisecond,
	maxDelay:    2 * time.Second,
}

// doWithRetry executes an HTTP request with exponential backoff + jitter.
// It retries only on transient failures: network errors, HTTP 429, and 5xx.
// The caller must still defer resp.Body.Close() on success.
func doWithRetry(client *http.Client, req *http.Request, cfg retryConfig) (*http.Response, error) {
	var lastErr error

	for attempt := range cfg.maxAttempts {
		if attempt > 0 {
			delay := backoffWithJitter(cfg.baseDelay, cfg.maxDelay, attempt)
			time.Sleep(delay)

			// Reset request body for retry (NewRequest stores the body via GetBody)
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = body
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue // network error → retry
		}

		if !isRetryable(resp.StatusCode) {
			return resp, nil // success or non-retryable error
		}

		// Retryable status code → close body and retry
		resp.Body.Close()
		lastErr = &retryableStatusError{statusCode: resp.StatusCode}
	}

	return nil, lastErr
}

// backoffWithJitter returns a delay with full jitter:
// delay = random(0, min(maxDelay, baseDelay * 2^attempt))
func backoffWithJitter(base, max time.Duration, attempt int) time.Duration {
	delay := base
	for range attempt {
		delay *= 2
		if delay > max {
			delay = max
			break
		}
	}
	// Full jitter: uniform random in [0, delay)
	return time.Duration(rand.Int64N(int64(delay)))
}

// isRetryable returns true for transient HTTP status codes.
func isRetryable(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

// retryableStatusError is returned when all retries are exhausted on a retryable status.
type retryableStatusError struct {
	statusCode int
}

func (e *retryableStatusError) Error() string {
	return http.StatusText(e.statusCode)
}
