package bdkeeper

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/drstein77/priceanalyzer/internal/models"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type Log interface {
	Info(string, ...zap.Field)
}

type BDKeeper struct {
	pool               *pgxpool.Pool
	log                Log
	userUpdateInterval func() string
}

func NewBDKeeper(dsn func() string, log Log) *BDKeeper {
	addr := dsn()
	if addr == "" {
		log.Info("database dsn is empty")
		return nil
	}

	config, err := pgxpool.ParseConfig(addr)
	if err != nil {
		log.Info("Unable to parse database DSN: ", zap.Error(err))
		return nil
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Info("Unable to connect to database: ", zap.Error(err))
		return nil
	}

	connConfig, err := pgx.ParseConfig(addr)
	if err != nil {
		log.Info("Unable to parse connection string: %v\n")
	}
	// Register the driver with the name pgx
	sqlDB := stdlib.OpenDB(*connConfig)

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		log.Info("Error getting driver: ", zap.Error(err))
		return nil
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Info("Error getting current directory: ", zap.Error(err))
	}

	// fix error test path
	mp := dir + "/migrations"
	var path string
	if _, err := os.Stat(mp); err != nil {
		path = "../../"
	} else {
		path = dir + "/"
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%smigrations", path),
		"postgres",
		driver)
	if err != nil {
		log.Info("Error creating migration instance: ", zap.Error(err))
		return nil
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		log.Info("Error while performing migration: ", zap.Error(err))
		return nil
	}

	log.Info("Connected!")

	return &BDKeeper{
		pool: pool,
		log:  log,
	}
}

func (kp *BDKeeper) Close() bool {
	if kp.pool != nil {
		kp.pool.Close()
		kp.log.Info("Database connection pool closed")
		return true
	}
	kp.log.Info("Attempted to close a nil database connection pool")
	return false
}

func (kp *BDKeeper) Ping(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	if err := kp.pool.Ping(ctx); err != nil {
		return false
	}

	return true
}

func (kp *BDKeeper) InsertProducts(products []models.Product) error {
	// Проверка подключения к базе
	if kp.pool == nil {
		return fmt.Errorf("database connection pool is nil")
	}

	// Начало транзакции
	tx, err := kp.pool.Begin(context.Background())
	if err != nil {
		kp.log.Info("Failed to begin transaction", zap.Error(err))
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(context.Background())
			kp.log.Info("Transaction rolled back due to an error")
		}
	}()

	// Подготовка запроса
	stmt := `
		INSERT INTO prices (id, name, category, price, create_date)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO NOTHING
	`
	batch := &pgx.Batch{}

	// Формирование пакета запросов
	for _, product := range products {
		batch.Queue(stmt, product.ID, product.Name, product.Category, product.Price, product.CreatedAt)
	}

	// Выполнение пакета
	br := tx.SendBatch(context.Background(), batch)
	defer br.Close()

	// Проверка ошибок выполнения запросов
	for i := 0; i < len(products); i++ {
		if _, err := br.Exec(); err != nil {
			kp.log.Info("Failed to execute batch query", zap.Error(err))
			return fmt.Errorf("failed to execute batch query: %w", err)
		}
	}

	// Коммит транзакции
	if err := tx.Commit(context.Background()); err != nil {
		kp.log.Info("Failed to commit transaction", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	kp.log.Info("Products successfully inserted into the database")
	return nil
}
