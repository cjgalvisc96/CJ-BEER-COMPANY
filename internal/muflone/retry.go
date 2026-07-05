package muflone

import (
	"errors"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Retry policy: optimistic-concurrency conflicts are EXPECTED under
// contention (two messages racing for one aggregate) and deserve patience;
// anything else gets a short fuse and then the dead-letter queue. Without
// this distinction, a hot aggregate could poison legitimate messages.
const (
	genericRetries     = 3
	concurrencyRetries = 12
	baseRetryInterval  = 50 * time.Millisecond
	maxRetryInterval   = time.Second
)

// retryMiddleware retries a failing handler with exponential backoff. The
// retry budget depends on the error: ErrConcurrency gets
// concurrencyRetries, everything else genericRetries — after which the
// error propagates to the poison queue.
func retryMiddleware(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		for attempt := 0; ; attempt++ {
			produced, err := h(msg)
			if err == nil {
				return produced, nil
			}
			budget := genericRetries
			if errors.Is(err, ErrConcurrency) {
				budget = concurrencyRetries
			}
			if attempt >= budget {
				return produced, err
			}
			select {
			case <-msg.Context().Done():
				return produced, err
			case <-time.After(retryBackoff(attempt)):
			}
		}
	}
}

func retryBackoff(attempt int) time.Duration {
	interval := baseRetryInterval << attempt
	if interval > maxRetryInterval || interval <= 0 {
		return maxRetryInterval
	}
	return interval
}
