package main

import (
	"sync/atomic"

	"github.com/gradusp/crispy-dummy/internal/app"
	"github.com/prometheus/client_golang/prometheus"
)

var appPromRegistry atomic.Value

func setupMetrics() error {
	ctx := app.Context()
	enabled, err := app.MetricsEnable.Maybe(ctx)
	if err != nil {
		return err
	}
	if enabled {
		appPromRegistry.Store(prometheus.NewRegistry())
	}
	return nil
}

//WhenHaveMetricsRegistry ...
func WhenHaveMetricsRegistry(f func(reg *prometheus.Registry)) {
	r, _ := appPromRegistry.Load().(*prometheus.Registry)
	if r != nil && f != nil {
		f(r)
	}
}
