package rest

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIPRateLimiterResetsWhenFull: the tracking map is bounded — past the
// cap it resets (forgiving, never blocking).
func TestIPRateLimiterResetsWhenFull(t *testing.T) {
	limiter := newIPRateLimiter(1, 1)
	for i := 0; i <= maxTrackedIPs; i++ {
		limiter.limiters["10.0."+strconv.Itoa(i/255)+"."+strconv.Itoa(i%255)+":"+strconv.Itoa(i)] = nil
	}

	assert.True(t, limiter.allow("fresh-client"), "reset map admits new clients")
	assert.Len(t, limiter.limiters, 1, "map was reset")
}
