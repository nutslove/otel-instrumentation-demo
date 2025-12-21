FROM python:3.12-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY manage.py .
COPY orderservice/ ./orderservice/
COPY orders/ ./orders/

# Create data directory
RUN mkdir -p /data

EXPOSE 8000

# Run migrations and start server with OpenTelemetry instrumentation
CMD ["sh", "-c", "python manage.py migrate --run-syncdb && opentelemetry-instrument gunicorn --bind 0.0.0.0:8000 --workers 1 orderservice.wsgi:application"]
