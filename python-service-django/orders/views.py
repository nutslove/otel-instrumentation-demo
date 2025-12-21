import json
import logging
import requests
from django.http import JsonResponse
from django.views.decorators.csrf import csrf_exempt
from django.views.decorators.http import require_http_methods
from .models import Order

logger = logging.getLogger(__name__)


def root(request):
    logger.info("Python Django service root endpoint called")
    return JsonResponse({"service": "python-django", "status": "running"})


@csrf_exempt
@require_http_methods(["GET", "POST"])
def orders_view(request):
    """Handle both GET (list orders) and POST (create order)"""
    if request.method == "GET":
        return get_orders(request)
    else:
        return create_order(request)


@csrf_exempt
def create_order(request):
    try:
        data = json.loads(request.body)
    except json.JSONDecodeError:
        return JsonResponse({"error": "Invalid JSON"}, status=400)

    user_id = data.get("user_id")
    product_name = data.get("product_name")
    quantity = data.get("quantity")

    if not all([user_id, product_name, quantity]):
        return JsonResponse(
            {"error": "Missing required fields: user_id, product_name, quantity"},
            status=400
        )

    logger.info(f"Creating order for user {user_id}")

    # Create order in database
    order = Order.objects.create(
        user_id=user_id,
        product_name=product_name,
        quantity=quantity,
        status="pending"
    )

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

    # If inventory is available, reserve it (Node.js -> Go)
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
                "message": f"Your order #{order.id} for {quantity}x {product_name} has been placed!",
                "type": "email"
            },
            timeout=10
        )
        notification_result = notification_response.json()
        logger.info(f"Notification sent: {notification_result}")
    except Exception as e:
        logger.error(f"Failed to send notification: {e}")
        notification_result = {"error": str(e)}

    logger.info(f"Order {order.id} created successfully")

    return JsonResponse({
        "order_id": order.id,
        "status": "pending",
        "inventory_check": inventory_result,
        "pricing": pricing_result,
        "notification": notification_result
    })


@require_http_methods(["GET"])
def get_orders(request):
    logger.info("Fetching all orders")

    orders = list(Order.objects.all().values(
        'id', 'user_id', 'product_name', 'quantity', 'status', 'created_at'
    ))

    # Convert datetime to string for JSON serialization
    for order in orders:
        order['created_at'] = order['created_at'].isoformat() if order['created_at'] else None

    logger.info(f"Retrieved {len(orders)} orders")
    return JsonResponse({"orders": orders})


def health(request):
    return JsonResponse({"status": "healthy"})


def intentional_error(request):
    logger.error("Intentional error triggered")
    return JsonResponse({"detail": "Intentional error for testing"}, status=500)


@csrf_exempt
@require_http_methods(["POST"])
def create_order_with_error(request):
    try:
        data = json.loads(request.body)
    except json.JSONDecodeError:
        return JsonResponse({"error": "Invalid JSON"}, status=400)

    user_id = data.get("user_id")
    product_name = data.get("product_name")
    quantity = data.get("quantity")

    if not all([user_id, product_name, quantity]):
        return JsonResponse(
            {"error": "Missing required fields: user_id, product_name, quantity"},
            status=400
        )

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

    return JsonResponse({
        "status": "error",
        "inventory_check": inventory_result,
        "pricing_error": pricing_result,
        "notification": notification_result,
        "message": "This order workflow intentionally includes errors across all services for testing"
    })
