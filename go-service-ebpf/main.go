package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

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
	// Initialize database
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

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
		log.Println("Go service root endpoint called")
		c.JSON(http.StatusOK, gin.H{
			"service": "go-gin-ebpf",
			"status":  "running",
		})
	})

	r.POST("/pricing/calculate", func(c *gin.Context) {
		var req PricingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("Invalid request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("Calculating pricing for %s (quantity: %d)", req.ProductName, req.Quantity)

		var unitPrice float64
		err := db.QueryRowContext(c.Request.Context(), "SELECT unit_price FROM pricing WHERE product_name = ?", req.ProductName).Scan(&unitPrice)
		if err != nil {
			log.Printf("Database error: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		totalPrice := unitPrice * float64(req.Quantity)
		log.Printf("Pricing calculated: %.2f", totalPrice)

		// Java serviceに通知を送信（Envoy egress検証用）
		// 環境変数JAVA_SERVICE_URLで接続先を切り替え
		// Envoy版: http://127.0.0.1:14318 (Envoy egress経由)
		// それ以外: http://java-service:8081 (直接)
		javaServiceURL := os.Getenv("JAVA_SERVICE_URL")
		if javaServiceURL == "" {
			javaServiceURL = "http://java-service:8081" // デフォルト
		}

		notificationData := map[string]interface{}{
			"recipient": "pricing-service@example.com",
			"message":   fmt.Sprintf("Price calculated: %s x %d = $%.2f", req.ProductName, req.Quantity, totalPrice),
			"type":      "pricing_notification",
		}

		notificationJSON, err := json.Marshal(notificationData)
		if err != nil {
			log.Printf("Failed to marshal notification data: %v", err)
		} else {
			notificationEndpoint := javaServiceURL + "/notifications/send"
			log.Printf("Sending notification to: %s", notificationEndpoint)

			client := &http.Client{}
			httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", notificationEndpoint, bytes.NewBuffer(notificationJSON))
			if err != nil {
				log.Printf("Failed to create notification request: %v", err)
			} else {
				httpReq.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(httpReq)
				if err != nil {
					log.Printf("Failed to send notification: %v", err)
				} else {
					defer resp.Body.Close()
					body, _ := io.ReadAll(resp.Body)
					log.Printf("Notification sent, response status: %d, body: %s", resp.StatusCode, string(body))
				}
			}
		}

		c.JSON(http.StatusOK, PricingResponse{
			ProductName: req.ProductName,
			UnitPrice:   unitPrice,
			Quantity:    req.Quantity,
			TotalPrice:  totalPrice,
		})
	})

	r.GET("/pricing", func(c *gin.Context) {
		log.Println("Fetching all pricing")

		rows, err := db.QueryContext(c.Request.Context(), "SELECT * FROM pricing")
		if err != nil {
			log.Printf("Database error: %v", err)
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

		log.Printf("Retrieved %d pricing items", len(pricing))

		c.JSON(http.StatusOK, gin.H{
			"pricing": pricing,
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.GET("/error", func(c *gin.Context) {
		log.Println("Intentional error triggered")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Intentional error for testing",
		})
	})

	r.POST("/pricing/calculate/error", func(c *gin.Context) {
		var req PricingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("Invalid request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("Intentional pricing error for %s", req.ProductName)

		var unitPrice float64
		err := db.QueryRowContext(c.Request.Context(), "SELECT unit_price FROM pricing WHERE product_name = ?", req.ProductName).Scan(&unitPrice)
		if err != nil {
			log.Printf("Database error: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		totalPrice := unitPrice * float64(req.Quantity)
		log.Printf("Pricing calculation error (intentional): %.2f", totalPrice)

		// Java serviceにエラー通知を送信（ヘッダー伝播なし - トレースが途切れることを示す）
		javaServiceURL := os.Getenv("JAVA_SERVICE_URL")
		if javaServiceURL == "" {
			javaServiceURL = "http://java-service:8081" // デフォルト
		}

		notificationData := map[string]interface{}{
			"recipient": "pricing-service@example.com",
			"message":   fmt.Sprintf("Pricing error: %s x %d = $%.2f (ERROR)", req.ProductName, req.Quantity, totalPrice),
			"type":      "pricing_error_notification",
		}

		notificationJSON, err := json.Marshal(notificationData)
		if err != nil {
			log.Printf("Failed to marshal notification data: %v", err)
		} else {
			notificationEndpoint := javaServiceURL + "/notifications/send"
			log.Printf("Sending error notification to: %s", notificationEndpoint)

			client := &http.Client{}
			httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", notificationEndpoint, bytes.NewBuffer(notificationJSON))
			if err != nil {
				log.Printf("Failed to create notification request: %v", err)
			} else {
				httpReq.Header.Set("Content-Type", "application/json")
				// NOTE: トレースヘッダーの伝播なし - Go → Javaでトレースが途切れる

				resp, err := client.Do(httpReq)
				if err != nil {
					log.Printf("Failed to send notification: %v", err)
				} else {
					defer resp.Body.Close()
					body, _ := io.ReadAll(resp.Body)
					log.Printf("Error notification sent, response status: %d, body: %s", resp.StatusCode, string(body))
				}
			}
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":        "Intentional pricing calculation error",
			"product_name": req.ProductName,
			"unit_price":   unitPrice,
			"quantity":     req.Quantity,
			"total_price":  totalPrice,
			"message":      "This is an intentional error for testing distributed tracing",
		})
	})

	// 新規: 価格計算後にJava serviceの通知エンドポイントを呼び出す
	// Envoy egressでのトレースヘッダー伝播を検証
	r.POST("/pricing/calculate/notify", func(c *gin.Context) {
		var req PricingRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("Invalid request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("Calculating pricing with notification for %s (quantity: %d)", req.ProductName, req.Quantity)

		// 価格計算
		var unitPrice float64
		err := db.QueryRowContext(c.Request.Context(), "SELECT unit_price FROM pricing WHERE product_name = ?", req.ProductName).Scan(&unitPrice)
		if err != nil {
			log.Printf("Database error: %v", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		totalPrice := unitPrice * float64(req.Quantity)
		log.Printf("Pricing calculated: %.2f", totalPrice)

		// Java serviceに通知を送信
		// 環境変数JAVA_SERVICE_URLで接続先を切り替え
		// Envoy版: http://127.0.0.1:14318 (Envoy egress経由)
		// それ以外: http://java-service:8081 (直接)
		javaServiceURL := os.Getenv("JAVA_SERVICE_URL")
		if javaServiceURL == "" {
			javaServiceURL = "http://java-service:8081" // デフォルト
		}

		notificationData := map[string]interface{}{
			"recipient": "pricing-service@example.com",
			"message":   fmt.Sprintf("Price calculated: %s x %d = $%.2f", req.ProductName, req.Quantity, totalPrice),
			"type":      "pricing_notification",
		}

		notificationJSON, err := json.Marshal(notificationData)
		if err != nil {
			log.Printf("Failed to marshal notification data: %v", err)
		} else {
			notificationEndpoint := javaServiceURL + "/notifications/send"
			log.Printf("Sending notification to: %s", notificationEndpoint)

			client := &http.Client{}
			httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", notificationEndpoint, bytes.NewBuffer(notificationJSON))
			if err != nil {
				log.Printf("Failed to create notification request: %v", err)
			} else {
				httpReq.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(httpReq)
				if err != nil {
					log.Printf("Failed to send notification: %v", err)
				} else {
					defer resp.Body.Close()
					body, _ := io.ReadAll(resp.Body)
					log.Printf("Notification sent, response status: %d, body: %s", resp.StatusCode, string(body))
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"product_name":      req.ProductName,
			"unit_price":        unitPrice,
			"quantity":          req.Quantity,
			"total_price":       totalPrice,
			"notification_sent": true,
			"java_service_url":  javaServiceURL,
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

	log.Println("Shutting down server...")
	if err := srv.Shutdown(nil); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}
