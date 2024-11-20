INSERT INTO orders_source(`order_id`, `customer_id`, `product_id`, `price`)
SELECT `order_id`, `customer_id`, `product_id`, `price` FROM examples.marketplace.orders;