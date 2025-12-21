from django.urls import path
from . import views

urlpatterns = [
    path('', views.root, name='root'),
    path('orders', views.orders_view, name='orders'),
    path('health', views.health, name='health'),
    path('error', views.intentional_error, name='error'),
    path('orders/error', views.create_order_with_error, name='create_order_error'),
]
