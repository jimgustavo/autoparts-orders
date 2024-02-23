-- Create a new table for orders
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    total NUMERIC NOT NULL,
    receipt_name VARCHAR(255),
    identification_number VARCHAR(255),
    phone_number VARCHAR(255),
    email VARCHAR(255),
    address VARCHAR(255),
    workshop_address VARCHAR(255),
    date VARCHAR(255),
    hour VARCHAR(255)
);

-- Create a table for order items
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id),
    name VARCHAR(255) NOT NULL,
    price NUMERIC NOT NULL
);
