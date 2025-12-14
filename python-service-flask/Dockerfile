FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY main.py .

# Create data directory
RUN mkdir -p /data

EXPOSE 8000

# Use OpenTelemetry automatic instrumentation with gunicorn
CMD ["opentelemetry-instrument", "gunicorn", "--bind", "0.0.0.0:8000", "--workers", "1", "main:app"]
