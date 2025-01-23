package dbkeeper

import (
	"context"
	"errors"
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

func (kp *DBKeeper) InsertProducts(ctx context.Context, products []models.Product) (*models.ProcessResponse, error) {
	if len(products) == 0 {
		return &models.ProcessResponse{}, nil
	}

	if kp.pool == nil {
		return nil, fmt.Errorf("database connection pool is nil")
	}

	tx, err := kp.pool.Begin(ctx)
	if err != nil {
		kp.log.Error("Failed to begin transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
				kp.log.Error("Failed to rollback transaction", zap.Error(rollbackErr))
			}
		}
	}()

	stmt := `INSERT INTO prices (name, category, price, create_date) VALUES ($1, $2, $3, $4)`
	batch := &pgx.Batch{}
	for _, product := range products {
		batch.Queue(stmt, product.Name, product.Category, product.Price, product.CreatedAt)
	}

	br := tx.SendBatch(ctx, batch)

	for range products {
		if _, execErr := br.Exec(); execErr != nil {
			err = fmt.Errorf("failed to execute batch query: %w", execErr)
			return nil, err
		}
	}

	if closeErr := br.Close(); closeErr != nil {
		kp.log.Error("Failed to close batch", zap.Error(closeErr))
	}

	var resp models.ProcessResponse
	statsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := tx.QueryRow(statsCtx, `
		SELECT COUNT(*), COUNT(DISTINCT category), COALESCE(SUM(price), 0)
		FROM prices
	`)

	if scanErr := row.Scan(&resp.TotalItems, &resp.TotalCategories, &resp.TotalPrice); scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return &models.ProcessResponse{}, nil
		}
		err = fmt.Errorf("failed to calculate stats: %w", scanErr)
		return nil, err
	}

	kp.log.Info("Committing transaction...")
	if commitErr := tx.Commit(ctx); commitErr != nil {
		err = fmt.Errorf("failed to commit transaction: %w", commitErr)
		return nil, err
	}

	kp.log.Info("Products successfully inserted, stats calculated.")
	return &resp, nil
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
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := kp.pool.Ping(ctx); err != nil {
		kp.log.Error("Database ping failed", zap.Error(err))
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
