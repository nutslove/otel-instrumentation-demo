import os
import sqlite3
import logging
from flask import Flask, request, jsonify
from flask_cors import CORS
import requests

# OpenTelemetry Flask instrumentation (explicit to ensure exemplars work with gunicorn)
from opentelemetry.instrumentation.flask import FlaskInstrumentor

# Configure logging - log format is set by OTEL_PYTHON_LOG_FORMAT environment variable
# Trace context injection is enabled by OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED
logging.basicConfig(
    level=logging.INFO
)
logger = logging.getLogger(__name__)

DB_PATH = "/data/orders.db"

app = Flask(__name__)
CORS(app)

# Explicitly instrument Flask app for proper exemplar support
# WSGI instrumentation should be disabled via OTEL_PYTHON_DISABLED_INSTRUMENTATIONS=wsgi
FlaskInstrumentor().instrument_app(app)

def init_db():
    """Initialize SQLite database"""
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS orders (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER NOT NULL,
            product_name TEXT NOT NULL,
            quantity INTEGER NOT NULL,
            status TEXT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    conn.commit()
    conn.close()

# Initialize database on startup
with app.app_context():
    init_db()

@app.route("/")
def root():
    logger.info("Python Flask service root endpoint called")
    return jsonify({"service": "python-flask", "status": "running"})

@app.route("/orders", methods=["POST"])
def create_order():
    data = request.get_json()

    if not data:
        return jsonify({"error": "No data provided"}), 400

    user_id = data.get("user_id")
    product_name = data.get("product_name")
    quantity = data.get("quantity")

    if not all([user_id, product_name, quantity]):
        return jsonify({"error": "Missing required fields: user_id, product_name, quantity"}), 400

    logger.info(f"Creating order for user {user_id}")

    # Database insert
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute(
        "INSERT INTO orders (user_id, product_name, quantity, status) VALUES (?, ?, ?, ?)",
        (user_id, product_name, quantity, "pending")
    )
    order_id = cursor.lastrowid
    conn.commit()
    conn.close()

    # Call Node.js service for inventory check
    try:
        response = requests.post(
            "http://nodejs-service:3000/inventory/check",
            json={"product_name": product_name, "quantity": quantity},
            timeout=10
        )
        inventory_result = response.json()
        logger.info(f"Inventory check result: {inventory_result}")
    except Exception as e:
        logger.error(f"Failed to check inventory: {e}")
        inventory_result = {"available": False, "error": str(e)}

    # If inventory is available, reserve it (Node.js → Go)
    pricing_result = None
    if inventory_result.get("available"):
        try:
            reserve_response = requests.post(
                "http://nodejs-service:3000/inventory/reserve",
                json={"product_name": product_name, "quantity": quantity},
                timeout=10
            )
            pricing_result = reserve_response.json()
            logger.info(f"Inventory reserved with pricing: {pricing_result}")
        except Exception as e:
            logger.error(f"Failed to reserve inventory: {e}")
            pricing_result = {"error": str(e)}

    # Send notification (Java)
    notification_result = None
    try:
        notification_response = requests.post(
            "http://java-service:8081/notifications/send",
            json={
                "recipient": f"user_{user_id}@example.com",
                "message": f"Your order #{order_id} for {quantity}x {product_name} has been placed!",
                "type": "email"
            },
            timeout=10
        )
        notification_result = notification_response.json()
        logger.info(f"Notification sent: {notification_result}")
    except Exception as e:
        logger.error(f"Failed to send notification: {e}")
        notification_result = {"error": str(e)}

    logger.info(f"Order {order_id} created successfully")

    return jsonify({
        "order_id": order_id,
        "status": "pending",
        "inventory_check": inventory_result,
        "pricing": pricing_result,
        "notification": notification_result
    })

@app.route("/orders", methods=["GET"])
def get_orders():
    logger.info("Fetching all orders")

    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM orders ORDER BY created_at DESC")
    orders = [dict(row) for row in cursor.fetchall()]
    conn.close()

    logger.info(f"Retrieved {len(orders)} orders")
    return jsonify({"orders": orders})

@app.route("/health")
def health():
    return jsonify({"status": "healthy"})

@app.route("/error")
def intentional_error():
    logger.error("Intentional error triggered")
    return jsonify({"detail": "Intentional error for testing"}), 500

@app.route("/orders/error", methods=["POST"])
def create_order_with_error():
    data = request.get_json()

    if not data:
        return jsonify({"error": "No data provided"}), 400

    user_id = data.get("user_id")
    product_name = data.get("product_name")
    quantity = data.get("quantity")

    if not all([user_id, product_name, quantity]):
        return jsonify({"error": "Missing required fields: user_id, product_name, quantity"}), 400

    logger.info(f"Creating order with intentional error for user {user_id}")

    # Call Node.js service for inventory check
    try:
        response = requests.post(
            "http://nodejs-service:3000/inventory/check",
            json={"product_name": product_name, "quantity": quantity},
            timeout=10
        )
        inventory_result = response.json()
        logger.info(f"Inventory check result: {inventory_result}")
    except Exception as e:
        logger.error(f"Failed to check inventory: {e}")
        inventory_result = {"available": False, "error": str(e)}

    # Call Node.js to reserve (which will call Go with error)
    pricing_result = None
    if inventory_result.get("available"):
        try:
            reserve_response = requests.post(
                "http://nodejs-service:3000/inventory/reserve/error",
                json={"product_name": product_name, "quantity": quantity},
                timeout=10
            )
            pricing_result = reserve_response.json()
            logger.info(f"Reserve response (with error): {pricing_result}")
        except Exception as e:
            logger.error(f"Failed to reserve inventory (expected error): {e}")
            pricing_result = {"error": str(e)}

    # Send notification (Java) even if error occurred
    notification_result = None
    try:
        notification_response = requests.post(
            "http://java-service:8081/notifications/send",
            json={
                "recipient": f"user_{user_id}@example.com",
                "message": f"Error occurred while processing order for {quantity}x {product_name}",
                "type": "email"
            },
            timeout=10
        )
        notification_result = notification_response.json()
        logger.info(f"Notification sent: {notification_result}")
    except Exception as e:
        logger.error(f"Failed to send notification: {e}")
        notification_result = {"error": str(e)}

    logger.error("Order workflow completed with errors")

    return jsonify({
        "status": "error",
        "inventory_check": inventory_result,
        "pricing_error": pricing_result,
        "notification": notification_result,
        "message": "This order workflow intentionally includes errors across all services for testing"
    })

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8000, debug=False)
