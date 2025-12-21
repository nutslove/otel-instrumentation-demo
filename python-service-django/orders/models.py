from django.db import models


class Order(models.Model):
    user_id = models.IntegerField()
    product_name = models.CharField(max_length=255)
    quantity = models.IntegerField()
    status = models.CharField(max_length=50, default='pending')
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        db_table = 'orders'
        ordering = ['-created_at']

    def __str__(self):
        return f"Order {self.id} - {self.product_name}"
