CREATE TABLE prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT,
    price NUMERIC(10, 2) NOT NULL,
    create_date TIMESTAMP NOT NULL DEFAULT NOW()
);