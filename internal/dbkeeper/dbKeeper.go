package dbkeeper

import (
	"context"
	"fmt"
	"time"

	"github.com/drstein77/priceanalyzer/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Log interface {
	Info(string, ...zap.Field)
	Error(string, ...zap.Field)
}

type DBKeeper struct {
	pool *pgxpool.Pool
	log  Log
}

func NewDBKeeper(ctx context.Context, dsn func() string, log Log) *DBKeeper {
	addr := dsn()
	if addr == "" {
		log.Error("database dsn is empty")
		return nil
	}

	config, err := pgxpool.ParseConfig(addr)
	if err != nil {
		log.Error("Unable to parse database DSN: ", zap.Error(err))
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Error("Unable to connect to database: ", zap.Error(err))
		return nil
	}

	log.Info("Connected!")

	return &DBKeeper{
		pool: pool,
		log:  log,
	}
}

func (kp *DBKeeper) InsertProducts(ctx context.Context, products []models.Product) (err error) {
	// Checking database connection
	if kp.pool == nil {
		return fmt.Errorf("database connection pool is nil")
	}

	// Beginning transaction
	tx, err := kp.pool.Begin(ctx)
	if err != nil {
		kp.log.Error("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Using deferred function to rollback transaction in case of an error
	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback(ctx)
			if rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
				// Logging rollback error without overriding the main error
				kp.log.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			} else {
				kp.log.Info("Transaction rolled back due to an error")
			}
		}
	}()

	// Preparing the query
	stmt := `
        INSERT INTO prices (id, name, category, price, create_date)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (id) DO NOTHING
    `
	batch := &pgx.Batch{}

	// Creating a batch of queries
	for _, product := range products {
		batch.Queue(stmt, product.ID, product.Name, product.Category, product.Price, product.CreatedAt)
	}

	// Executing the batch
	br := tx.SendBatch(ctx, batch)

	// Checking for errors in query execution
	for i := 0; i < len(products); i++ {
		if _, execErr := br.Exec(); execErr != nil {
			br.Close() // Closing the batch before returning an error
			err = fmt.Errorf("failed to execute batch query: %w", execErr)
			return err
		}
	}

	// Closing the batch after processing all queries
	if closeErr := br.Close(); closeErr != nil {
		err = fmt.Errorf("failed to close batch results: %w", closeErr)
		return err
	}

	// Committing the transaction
	if commitErr := tx.Commit(ctx); commitErr != nil {
		err = fmt.Errorf("failed to commit transaction: %w", commitErr)
		return err
	}

	kp.log.Info("Products successfully inserted into the database")
	return nil
}

func (kp *DBKeeper) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	// Checking database connection
	if kp.pool == nil {
		return nil, fmt.Errorf("database connection pool is nil")
	}

	// SQL query to fetch all data from the table
	query := `
		SELECT id, name, category, price, create_date
		FROM prices
	`

	// Executing the query
	rows, err := kp.pool.Query(ctx, query)
	if err != nil {
		kp.log.Error("Failed to execute query", zap.Error(err))
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Reading data
	var products []models.Product
	for rows.Next() {
		var product models.Product
		err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Category,
			&product.Price,
			&product.CreatedAt,
		)
		if err != nil {
			kp.log.Error("Failed to scan row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		products = append(products, product)
	}

	// Checking for errors during iteration
	if rows.Err() != nil {
		kp.log.Error("Error occurred during rows iteration", zap.Error(rows.Err()))
		return nil, fmt.Errorf("error during rows iteration: %w", rows.Err())
	}

	kp.log.Info("Successfully retrieved all products", zap.Int("count", len(products)))
	return products, nil
}

func (kp *DBKeeper) Ping(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	if err := kp.pool.Ping(ctx); err != nil {
		return false
	}

	return true
}

func (kp *DBKeeper) Close() bool {
	if kp.pool != nil {
		kp.pool.Close()
		kp.log.Info("Database connection pool closed")
		return true
	}
	kp.log.Info("Attempted to close a nil database connection pool")
	return false
}
