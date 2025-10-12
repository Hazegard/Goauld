package http

import (
	"time"
)

// Leaky bucket rate limiter. Lets you do rate units per second, in a burst of
// up to max. The rate increases linearly at a rate of rateRateOfIncrease units
// per second.
type RateLimiter struct {
	rate               float64
	max                float64
	rateRateOfIncrease float64
	cur                float64 // < 0 means limited, >= 0 means free to act
	lastUpdate         time.Time
}

// NewRateLimiter creates a new RateLimiter with the given parameters.
func NewRateLimiter(now time.Time, rate, max, rateRateOfIncrease float64) RateLimiter {
	return RateLimiter{
		rate:               rate,
		max:                max,
		rateRateOfIncrease: rateRateOfIncrease,
		lastUpdate:         now,
	}
}

// update refills the bucket for the amount of time that has passed since the
// last update at the current rate, up to max.
func (rl *RateLimiter) update(now time.Time) {
	if now.Before(rl.lastUpdate) {
		return
	}
	elapsed := now.Sub(rl.lastUpdate).Seconds()
	rl.lastUpdate = now
	// Replenish the bucket capacity.
	rl.cur = rl.cur + rl.rate*elapsed
	if rl.cur > rl.max {
		rl.cur = rl.max
	}
	// Increase the rate.
	rl.rate += rl.rateRateOfIncrease * elapsed
}

// IsLimited returns two values: a bool indicating whether the limiter is
// currently limiting, and a time.Duration indicating an amount of time after
// which it will be free to act again.
func (rl *RateLimiter) IsLimited(now time.Time) (bool, time.Duration) {
	rl.update(now)
	if rl.cur < 0.0 {
		return true, time.Duration(-rl.cur / rl.rate * 1e9)
	} else {
		return false, 0
	}
}

// Take removes an amount of capacity from the bucket. If this causes the
// capacity to go negative, the limiter will start limiting.
func (rl *RateLimiter) Take(now time.Time, amount float64) {
	rl.update(now)
	rl.cur -= amount
}

func (rl *RateLimiter) MultiplicativeDecrease(now time.Time, factor float64) {
	rl.update(now)
	rl.rate *= factor
	rl.cur = 0.0
}
