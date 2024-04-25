package work_queue

import (
	"math"
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/services"
)

// ExponentialBackoff returns a backoff algorithm function that will retry with an interval that doubles
// with each retry until it gets up to the specified maximum interval. This will continue until the specified
// maximum number of attempts have occurred.
func ExponentialBackoff(maxAttempts int, initialRetryInterval, maxRetryInterval time.Duration) services.BackoffAlgorithm {
	return func(attemptsSoFar int, lastAttemptAt time.Time, workItem *models.WorkItem) *time.Time {
		if attemptsSoFar >= maxAttempts {
			return nil // No more retries
		}

		// Calculate retry interval
		var interval time.Duration
		doublingCount := math.Min(float64(attemptsSoFar-1), 30) // avoid overflow; doubling 30 times is enough
		multiple := math.Pow(2, doublingCount)
		unboundedInterval := float64(initialRetryInterval) * multiple
		if unboundedInterval < float64(maxRetryInterval) {
			interval = time.Duration(unboundedInterval)
		} else {
			interval = maxRetryInterval
		}

		notBefore := lastAttemptAt.Add(interval)
		return &notBefore
	}
}

// LinearBackoff returns a BackoffAlgorithm function that will retry with a fixed interval, up to the specified
// maximum number of attempts.
func LinearBackoff(maxAttempts int, retryInterval time.Duration) services.BackoffAlgorithm {
	return func(attemptsSoFar int, lastAttemptAt time.Time, workItem *models.WorkItem) *time.Time {
		if attemptsSoFar >= maxAttempts {
			return nil // No more retries
		}
		notBefore := lastAttemptAt.Add(retryInterval)
		return &notBefore
	}
}

// RetryOnce returns a BackoffAlgorithm function that will retry only once, i.e. a maximum of 2 attempts.
func RetryOnce(retryInterval time.Duration) services.BackoffAlgorithm {
	return LinearBackoff(2, retryInterval)
}

// NoRetry returns a BackoffAlgorithm function that will not retry at all.
func NoRetry() services.BackoffAlgorithm {
	return func(attemptsSoFar int, lastAttemptAt time.Time, workItem *models.WorkItem) *time.Time {
		return nil // never retry
	}
}
