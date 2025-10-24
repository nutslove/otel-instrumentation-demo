package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/log/global"
	otlog "go.opentelemetry.io/otel/log"
)

var (
	db     *sql.DB
	tracer trace.Tracer
	logger otlog.Logger
)

type PricingRequest struct {
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}

type PricingResponse struct {
	ProductName string  `json:"product_name"`
	UnitPrice   float64 `json:"unit_price"`
	Quantity    int     `json:"quantity"`
	TotalPrice  float64 `json:"total_price"`
}

func initTelemetry(ctx context.Context) (func(), error) {
	// Resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("go-gin-service"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Trace exporter
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("adot-collector:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Trace provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tracer = tracerProvider.Tracer("go-service-tracer")

	// Metrics exporter
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint("adot-collector:4317"),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	// Log exporter
	logExporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint("adot-collector:4317"),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	// Logger provider
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(loggerProvider)
	logger = loggerProvider.Logger("go-service-logger")

	// Cleanup function
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tracerProvider.Shutdown(ctx)
		meterProvider.Shutdown(ctx)
		loggerProvider.Shutdown(ctx)
	}

	return cleanup, nil
}

func emitLog(ctx context.Context, severity otlog.Severity, message string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	spanCtx := span.SpanContext()

	// Convert attribute.KeyValue to otlog.KeyValue
	logAttrs := make([]otlog.KeyValue, len(attrs)+2)
	for i, attr := range attrs {
		logAttrs[i] = otlog.String(string(attr.Key), attr.Value.AsString())
	}
	logAttrs[len(attrs)] = otlog.String("trace_id", spanCtx.TraceID().String())
	logAttrs[len(attrs)+1] = otlog.String("span_id", spanCtx.SpanID().String())

	var record otlog.Record
	record.SetTimestamp(time.Now())
	record.SetSeverity(severity)
	record.SetBody(otlog.StringValue(message))
	record.AddAttributes(logAttrs...)

	logger.Emit(ctx, record)
}

func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "/data/pricing.db")
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS pricing (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_name TEXT NOT NULL UNIQUE,
			unit_price REAL NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// Insert sample data
	_, err = db.Exec(`INSERT OR IGNORE INTO pricing (id, product_name, unit_price) VALUES (1, 'Laptop', 999.99)`)
	_, err = db.Exec(`INSERT OR IGNORE INTO pricing (id, product_name, unit_price) VALUES (2, 'Mouse', 29.99)`)
	_, err = db.Exec(`INSERT OR IGNORE INTO pricing (id, product_name, unit_price) VALUES (3, 'Keyboard', 79.99)`)

	return err
}

func main() {
	ctx := context.Background()

	// Initialize telemetry
	cleanup, err := initTelemetry(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer cleanup()

	// Initialize database
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware("go-gin-service"))

	// CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "*")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()

		emitLog(ctx, otlog.SeverityInfo, fmt.Sprintf("Go service root endpoint called - trace_id: %s", traceID))

		c.JSON(http.StatusOK, gin.H{
			"service": "go-gin",
			"status":  "running",
		})
	})

	r.POST("/pricing/calculate", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()

		var req PricingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			emitLog(ctx, otlog.SeverityError, fmt.Sprintf("Invalid request: %v - trace_id: %s", err, traceID))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		emitLog(ctx, otlog.SeverityInfo, fmt.Sprintf("Calculating pricing for %s - trace_id: %s", req.ProductName, traceID),
			attribute.String("product.name", req.ProductName),
			attribute.Int("quantity", req.Quantity),
		)

		// Database query with span
		dbCtx, dbSpan := tracer.Start(ctx, "db_select_pricing",
			trace.WithAttributes(
				attribute.String("db.operation", "select"),
				attribute.String("db.table", "pricing"),
				attribute.String("product.name", req.ProductName),
			),
		)

		var unitPrice float64
		err := db.QueryRowContext(dbCtx, "SELECT unit_price FROM pricing WHERE product_name = ?", req.ProductName).Scan(&unitPrice)
		dbSpan.End()

		if err != nil {
			emitLog(ctx, otlog.SeverityError, fmt.Sprintf("Database error: %v - trace_id: %s", err, traceID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		totalPrice := unitPrice * float64(req.Quantity)

		emitLog(ctx, otlog.SeverityInfo, fmt.Sprintf("Pricing calculated: %.2f - trace_id: %s", totalPrice, traceID),
			attribute.Float64("unit.price", unitPrice),
			attribute.Float64("total.price", totalPrice),
		)

		c.JSON(http.StatusOK, PricingResponse{
			ProductName: req.ProductName,
			UnitPrice:   unitPrice,
			Quantity:    req.Quantity,
			TotalPrice:  totalPrice,
		})
	})

	r.GET("/pricing", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()

		emitLog(ctx, otlog.SeverityInfo, fmt.Sprintf("Fetching all pricing - trace_id: %s", traceID))

		dbCtx, dbSpan := tracer.Start(ctx, "db_select_all_pricing",
			trace.WithAttributes(
				attribute.String("db.operation", "select"),
				attribute.String("db.table", "pricing"),
			),
		)

		rows, err := db.QueryContext(dbCtx, "SELECT * FROM pricing")
		dbSpan.End()

		if err != nil {
			emitLog(ctx, otlog.SeverityError, fmt.Sprintf("Database error: %v - trace_id: %s", err, traceID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var pricing []map[string]interface{}
		for rows.Next() {
			var id int
			var productName string
			var unitPrice float64
			var updatedAt string
			rows.Scan(&id, &productName, &unitPrice, &updatedAt)
			pricing = append(pricing, map[string]interface{}{
				"id":           id,
				"product_name": productName,
				"unit_price":   unitPrice,
				"updated_at":   updatedAt,
			})
		}

		emitLog(ctx, otlog.SeverityInfo, fmt.Sprintf("Retrieved %d pricing items - trace_id: %s", len(pricing), traceID))

		c.JSON(http.StatusOK, gin.H{
			"pricing": pricing,
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.GET("/error", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()

		emitLog(ctx, otlog.SeverityError, fmt.Sprintf("Intentional error triggered - trace_id: %s", traceID))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Intentional error for testing",
		})
	})

	r.POST("/pricing/calculate/error", func(c *gin.Context) {
		ctx := c.Request.Context()
		span := trace.SpanFromContext(ctx)
		traceID := span.SpanContext().TraceID().String()

		var req PricingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			emitLog(ctx, otlog.SeverityError, fmt.Sprintf("Invalid request: %v - trace_id: %s", err, traceID))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		emitLog(ctx, otlog.SeverityError, fmt.Sprintf("Intentional pricing error for %s - trace_id: %s", req.ProductName, traceID),
			attribute.String("product.name", req.ProductName),
			attribute.Int("quantity", req.Quantity),
		)

		// Simulate pricing calculation but return error
		dbCtx, dbSpan := tracer.Start(ctx, "db_select_pricing_error",
			trace.WithAttributes(
				attribute.String("db.operation", "select"),
				attribute.String("db.table", "pricing"),
				attribute.String("product.name", req.ProductName),
			),
		)

		var unitPrice float64
		err := db.QueryRowContext(dbCtx, "SELECT unit_price FROM pricing WHERE product_name = ?", req.ProductName).Scan(&unitPrice)
		dbSpan.End()

		if err != nil {
			emitLog(ctx, otlog.SeverityError, fmt.Sprintf("Database error: %v - trace_id: %s", err, traceID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		totalPrice := unitPrice * float64(req.Quantity)

		emitLog(ctx, otlog.SeverityError, fmt.Sprintf("Pricing calculation error (intentional): %.2f - trace_id: %s", totalPrice, traceID))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":        "Intentional pricing calculation error",
			"product_name": req.ProductName,
			"unit_price":   unitPrice,
			"quantity":     req.Quantity,
			"total_price":  totalPrice,
			"message":      "This is an intentional error for testing distributed tracing",
		})
	})

	// Start server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Go service listening on port 8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
