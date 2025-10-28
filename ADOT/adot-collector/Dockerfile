FROM public.ecr.aws/aws-observability/aws-otel-collector:latest

# Copy custom configuration file to standard path
COPY otel-collector-config.yaml /etc/otel-config.yaml

# Port 4317 for OTLP/gRPC
# Port 4318 for OTLP/HTTP
EXPOSE 4317 4318

# Override the default command to use our custom config
CMD ["--config=/etc/otel-config.yaml"]
