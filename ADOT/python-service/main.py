import os
import sqlite3
import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel
import httpx

# Configure logging - log format is set by OTEL_PYTHON_LOG_FORMAT environment variable
# Trace context injection is enabled by OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED
logging.basicConfig(
    level=logging.INFO
)
logger = logging.getLogger(__name__)

DB_PATH = "/data/orders.db"

# Create instrumented httpx client
httpx_client = httpx.AsyncClient()

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

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    init_db()
    yield
    # Shutdown
    await httpx_client.aclose()

app = FastAPI(title="Python Order Service", lifespan=lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

class Order(BaseModel):
    user_id: int
    product_name: str
    quantity: int

@app.get("/")
async def root():
    logger.info("Python service root endpoint called")
    return {"service": "python-fastapi", "status": "running"}

@app.post("/orders")
async def create_order(order: Order):
    logger.info(f"Creating order for user {order.user_id}")

    # Database insert
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute(
        "INSERT INTO orders (user_id, product_name, quantity, status) VALUES (?, ?, ?, ?)",
        (order.user_id, order.product_name, order.quantity, "pending")
    )
    order_id = cursor.lastrowid
    conn.commit()
    conn.close()

    # Call Node.js service for inventory check
    try:
        response = await httpx_client.post(
            "http://localhost:3000/inventory/check",
            json={"product_name": order.product_name, "quantity": order.quantity}
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
            reserve_response = await httpx_client.post(
                "http://localhost:3000/inventory/reserve",
                json={"product_name": order.product_name, "quantity": order.quantity}
            )
            pricing_result = reserve_response.json()
            logger.info(f"Inventory reserved with pricing: {pricing_result}")
        except Exception as e:
            logger.error(f"Failed to reserve inventory: {e}")
            pricing_result = {"error": str(e)}

    # Send notification (Java)
    notification_result = None
    try:
        notification_response = await httpx_client.post(
            "http://localhost:8081/notifications/send",
            json={
                "recipient": f"user_{order.user_id}@example.com",
                "message": f"Your order #{order_id} for {order.quantity}x {order.product_name} has been placed!",
                "type": "email"
            }
        )
        notification_result = notification_response.json()
        logger.info(f"Notification sent: {notification_result}")
    except Exception as e:
        logger.error(f"Failed to send notification: {e}")
        notification_result = {"error": str(e)}

    logger.info(f"Order {order_id} created successfully")

    return {
        "order_id": order_id,
        "status": "pending",
        "inventory_check": inventory_result,
        "pricing": pricing_result,
        "notification": notification_result
    }

@app.get("/orders")
async def get_orders():
    logger.info("Fetching all orders")

    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    cursor.execute("SELECT * FROM orders ORDER BY created_at DESC")
    orders = [dict(row) for row in cursor.fetchall()]
    conn.close()

    logger.info(f"Retrieved {len(orders)} orders")
    return {"orders": orders}

@app.get("/health")
async def health():
    return {"status": "healthy"}

@app.get("/error")
async def intentional_error():
    logger.error("Intentional error triggered")
    raise HTTPException(status_code=500, detail="Intentional error for testing")

@app.post("/orders/error")
async def create_order_with_error(order: Order):
    logger.info(f"Creating order with intentional error for user {order.user_id}")

    # Call Node.js service for inventory check
    try:
        response = await httpx_client.post(
            "http://localhost:3000/inventory/check",
            json={"product_name": order.product_name, "quantity": order.quantity}
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
            reserve_response = await httpx_client.post(
                "http://localhost:3000/inventory/reserve/error",
                json={"product_name": order.product_name, "quantity": order.quantity}
            )
            pricing_result = reserve_response.json()
            logger.info(f"Reserve response (with error): {pricing_result}")
        except Exception as e:
            logger.error(f"Failed to reserve inventory (expected error): {e}")
            pricing_result = {"error": str(e)}

    # Send notification (Java) even if error occurred
    notification_result = None
    try:
        notification_response = await httpx_client.post(
            "http://localhost:8081/notifications/send",
            json={
                "recipient": f"user_{order.user_id}@example.com",
                "message": f"Error occurred while processing order for {order.quantity}x {order.product_name}",
                "type": "email"
            }
        )
        notification_result = notification_response.json()
        logger.info(f"Notification sent: {notification_result}")
    except Exception as e:
        logger.error(f"Failed to send notification: {e}")
        notification_result = {"error": str(e)}

    logger.error("Order workflow completed with errors")

    return {
        "status": "error",
        "inventory_check": inventory_result,
        "pricing_error": pricing_result,
        "notification": notification_result,
        "message": "This order workflow intentionally includes errors across all services for testing"
    }
