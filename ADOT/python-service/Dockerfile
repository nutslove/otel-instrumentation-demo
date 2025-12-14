FROM python:3.11-slim

WORKDIR /app

# Install curl for health checks
RUN apt-get update && apt-get install -y curl && rm -rf /var/lib/apt/lists/*

# Install other dependencies first
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# # Copy and install patched packages AFTER requirements.txt to ensure they override official packages
# COPY otel-python-patched-packages /tmp/otel-python-patched-packages
# RUN pip install --no-cache-dir --force-reinstall --no-deps /tmp/otel-python-patched-packages/asgi /tmp/otel-python-patched-packages/httpx

# # Clean up
# RUN rm -rf /tmp/otel-python-patched-packages

COPY main.py .

# Create data directory
RUN mkdir -p /data

EXPOSE 8000

# Use OpenTelemetry automatic instrumentation
CMD ["opentelemetry-instrument", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
