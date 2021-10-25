package main

import (
	"context"

	"github.com/gradusp/crispy-dummy/internal/api/dummy"
	"github.com/gradusp/crispy-dummy/internal/app"
	"github.com/gradusp/go-platform/server"
	"github.com/gradusp/go-platform/server/interceptors"
	serverPrometheusMetrics "github.com/gradusp/go-platform/server/metrics/prometheus"
	"github.com/prometheus/client_golang/prometheus"
)

func setupServer(ctx context.Context) (*server.APIServer, error) {
	netInterfaceName, err := app.AnnounceInterfaceName.Maybe(ctx)
	if err != nil {
		return nil, err
	}
	var service server.APIService
	service, err = dummy.NewAnnounceService(ctx, netInterfaceName)
	if err != nil {
		return nil, err
	}

	var doc *server.SwaggerSpec
	if doc, err = dummy.GetSwaggerDocs(); err != nil {
		return nil, err
	}

	opts := []server.APIServerOption{
		server.WithServices(service),
		server.WithDocs(doc, ""),
	}

	//если есть регистр Прометеуса то - подклчим метрики
	WhenHaveMetricsRegistry(func(reg *prometheus.Registry) {
		pm := serverPrometheusMetrics.NewMetrics(
			serverPrometheusMetrics.WithSubsystem("grpc"),
			serverPrometheusMetrics.WithNamespace("server"),
		)
		if err = reg.Register(pm); err == nil {
			recovery := interceptors.NewRecovery(
				interceptors.RecoveryWithObservers(pm.PanicsObserver()), //подключаем prometheus счетчик паник
			)
			//подключаем prometheus метрики
			opts = append(opts, server.WithRecovery(recovery))
			opts = append(opts, server.WithStatsHandlers(pm.StatHandlers()...))
		}
	})
	if err != nil {
		return nil, err
	}
	var srv *server.APIServer
	srv, err = server.NewAPIServer(opts...)
	if err != nil {
		return nil, err
	}
	return srv, nil
}
