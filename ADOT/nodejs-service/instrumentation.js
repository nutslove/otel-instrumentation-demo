const { NodeSDK, metrics, logs } = require('@opentelemetry/sdk-node');
const { OTLPMetricExporter } = require('@opentelemetry/exporter-metrics-otlp-grpc');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-grpc');
const { OTLPLogExporter } = require('@opentelemetry/exporter-logs-otlp-grpc');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { RuntimeNodeInstrumentation } = require('@opentelemetry/instrumentation-runtime-node');

const sdk = new NodeSDK({
  traceExporter: new OTLPTraceExporter(),
  metricReader: new metrics.PeriodicExportingMetricReader({
    exporter: new OTLPMetricExporter(),
  }),
  logRecordProcessor: new logs.SimpleLogRecordProcessor(
    new OTLPLogExporter()
  ),
  instrumentations: [
    getNodeAutoInstrumentations(),
    new RuntimeNodeInstrumentation(),
  ],
});

sdk.start();

// Graceful shutdown
process.on('SIGTERM', async () => {
  try {
    await sdk.shutdown();
    console.log('OpenTelemetry SDK shut down successfully');
  } catch (error) {
    console.error('Error shutting down OpenTelemetry SDK', error);
  }
});