FROM public.ecr.aws/aws-observability/aws-otel-collector:latest

# Copy custom configuration file
COPY otel-collector-config.yaml /etc/otelcol-contrib/config.yaml

# The collector will use this config file by default
# Port 4317 for OTLP/gRPC
# Port 4318 for OTLP/HTTP
EXPOSE 4317 4318
