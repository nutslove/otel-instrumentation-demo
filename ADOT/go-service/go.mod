module go-pricing-service

go 1.23

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/mattn/go-sqlite3 v1.14.19
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.47.0
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/bridge/opentracing v1.22.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v0.45.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.22.0
	go.opentelemetry.io/otel/sdk v1.22.0
	go.opentelemetry.io/otel/sdk/metric v1.22.0
	go.opentelemetry.io/otel/trace v1.22.0
)
