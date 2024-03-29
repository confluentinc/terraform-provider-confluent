INSERT INTO orders_source(
    order_id,
    customer_id,
    product_id,
    price
)
SELECT * FROM examples.marketplace.orders;