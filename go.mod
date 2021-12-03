module github.com/gradusp/crispy-dummy

go 1.16

require (
	github.com/gradusp/go-platform v0.0.4-dev
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.6.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cast v1.4.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/vishvananda/netlink v1.1.0
	go.opentelemetry.io/otel v1.0.0-RC3
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.0.0-RC3
	go.opentelemetry.io/otel/sdk v1.0.0-RC3
	go.opentelemetry.io/otel/trace v1.0.0-RC3
	go.uber.org/zap v1.17.0
	golang.org/x/sys v0.0.0-20210902050250-f475640dd07b // indirect
	google.golang.org/genproto v0.0.0-20210903162649-d08c68adba83
	google.golang.org/grpc v1.40.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.27.1
)

//replace github.com/gradusp/go-platform v0.0.4-dev => ../go-platform
