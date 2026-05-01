package health

import (
	"context"
	"log/slog"
	"obsidian-auth/pkg/lib/log/sl"
	"sync"
	"time"

	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Checker struct {
	srv      *health.Server
	log      *slog.Logger
	pg       Pinger
	redis    Pinger
	interval time.Duration
	timeout  time.Duration
}

func New(log *slog.Logger, pg, redis Pinger, interval, timeout time.Duration) *Checker {
	return &Checker{
		srv:      health.NewServer(),
		log:      log,
		pg:       pg,
		redis:    redis,
		interval: interval,
		timeout:  timeout,
	}
}

func (c *Checker) Server() *health.Server {
	return c.srv
}

func (c *Checker) Run(ctx context.Context) {
	c.setInitial()

	t := time.NewTicker(c.interval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			c.updateReadiness(ctx)
		case <-ctx.Done():
			c.setNotServing()
			return
		}
	}
}

func (c *Checker) setInitial() {
	c.srv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	c.srv.SetServingStatus("liveness", grpc_health_v1.HealthCheckResponse_SERVING)
	c.srv.SetServingStatus("readiness", grpc_health_v1.HealthCheckResponse_UNKNOWN)
}

func (c *Checker) setNotServing() {
	c.srv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	c.srv.SetServingStatus("liveness", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	c.srv.SetServingStatus("readiness", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
}

func (c *Checker) updateReadiness(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	errors := c.check(checkCtx, c.pg, c.redis)
	cancel()

	if len(errors) > 0 {
		c.srv.SetServingStatus("readiness", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

		for _, err := range errors {
			c.log.Error("Readiness status error", sl.Err(err))
		}
	} else {
		c.srv.SetServingStatus("readiness", grpc_health_v1.HealthCheckResponse_SERVING)
	}
}

func (c *Checker) check(ctx context.Context, pingers ...Pinger) []error {
	errorsChan := make(chan error, len(pingers))

	var wg sync.WaitGroup
	wg.Add(len(pingers))

	for _, p := range pingers {
		go func(p Pinger) {
			defer wg.Done()
			errorsChan <- p.Ping(ctx)
		}(p)
	}

	go func() {
		wg.Wait()
		close(errorsChan)
	}()

	var errors []error

	for err := range errorsChan {
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}
