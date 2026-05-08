package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/sony/gobreaker"
)

// cbState mirrors gobreaker.State for the Prometheus metric.
var cbStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "redis_circuit_breaker_state",
	Help: "Circuit breaker state: 0=CLOSED 1=OPEN 2=HALF-OPEN",
}, []string{"instance"})

// CircuitBreakerClient wraps redis-persistent reads behind a Sony gobreaker.
// On OPEN state, callers receive ErrCircuitOpen and must fall back to PostgreSQL.
// (PRD §5.4 Redis Circuit Breaker Fallback)
type CircuitBreakerClient struct {
	client *redis.Client
	cb     *gobreaker.CircuitBreaker
	name   string
}

// ErrCircuitOpen is returned when the circuit breaker is OPEN.
var ErrCircuitOpen = fmt.Errorf("redis circuit breaker open — use fallback")

func NewCircuitBreakerClient(client *redis.Client, name string) *CircuitBreakerClient {
	cbSettings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,                // max requests in HALF-OPEN
		Interval:    60 * time.Second, // clears counts after this interval
		Timeout:     30 * time.Second, // stays OPEN for this long before probing
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip after 5 consecutive failures OR >50% failure rate with ≥10 requests
			if counts.ConsecutiveFailures >= 5 {
				return true
			}
			if counts.Requests >= 10 {
				failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
				return failureRatio > 0.5
			}
			return false
		},
		OnStateChange: func(n string, from, to gobreaker.State) {
			val := float64(to) // CLOSED=0, OPEN=1, HALF-OPEN=2
			cbStateGauge.WithLabelValues(n).Set(val)
		},
	}
	cb := gobreaker.NewCircuitBreaker(cbSettings)

	return &CircuitBreakerClient{client: client, cb: cb, name: name}
}

// Get wraps redis GET with circuit breaker protection.
func (c *CircuitBreakerClient) Get(ctx context.Context, key string) (string, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		val, err := c.client.Get(ctx, key).Result()
		if err == redis.Nil {
			return "", nil // cache miss is not a circuit failure
		}
		return val, err
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return "", ErrCircuitOpen
		}
		return "", err
	}
	str, _ := result.(string)
	return str, nil
}

// Set wraps redis SET with circuit breaker protection.
func (c *CircuitBreakerClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		return nil, c.client.Set(ctx, key, value, ttl).Err()
	})
	if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
		return ErrCircuitOpen
	}
	return err
}

// ZRevRangeWithScores wraps ZREVRANGE WITHSCORES with circuit breaker.
func (c *CircuitBreakerClient) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.ZRevRangeWithScores(ctx, key, start, stop).Result()
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, ErrCircuitOpen
		}
		return nil, err
	}
	return result.([]redis.Z), nil
}

// HGetAll wraps HGETALL with circuit breaker.
func (c *CircuitBreakerClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	result, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.HGetAll(ctx, key).Result()
	})
	if err != nil {
		if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
			return nil, ErrCircuitOpen
		}
		return nil, err
	}
	return result.(map[string]string), nil
}

// ZIncrBy wraps ZINCRBY with circuit breaker — used by worker to update leaderboards.
func (c *CircuitBreakerClient) ZIncrBy(ctx context.Context, key string, increment float64, member string) error {
	_, err := c.cb.Execute(func() (interface{}, error) {
		return c.client.ZIncrBy(ctx, key, increment, member).Result()
	})
	if err == gobreaker.ErrOpenState || err == gobreaker.ErrTooManyRequests {
		return ErrCircuitOpen
	}
	return err
}

// State returns the current circuit breaker state as a string.
func (c *CircuitBreakerClient) State() string {
	switch c.cb.State() {
	case gobreaker.StateClosed:
		return "CLOSED"
	case gobreaker.StateOpen:
		return "OPEN"
	default:
		return "HALF-OPEN"
	}
}

// Unwrap returns the raw redis client (for operations that bypass the CB, e.g. SetNX for locks).
func (c *CircuitBreakerClient) Unwrap() *redis.Client {
	return c.client
}
