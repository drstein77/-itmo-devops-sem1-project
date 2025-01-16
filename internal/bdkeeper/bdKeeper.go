package bdkeeper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/drstein77/priceanalyzer/internal/models"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // registers a migrate driver.
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

type Log interface {
	Info(string, ...zap.Field)
}

type BDKeeper struct {
	pool *pgxpool.Pool
	log  Log
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

	migrationsDir, err := findMigrationsDir()
	if err != nil {
		log.Info("Ошибка поиска папки 'migrations': ", zap.Error(err))
		return nil
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsDir),
		"postgres",
		driver)
	if err != nil {
		log.Info("Ошибка создания экземпляра миграции: ", zap.Error(err))
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

func (kp *BDKeeper) InsertProducts(products []models.Product) (err error) {
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

	// Используем deferred функцию для отката транзакции в случае ошибки
	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback(context.Background())
			if rollbackErr != nil && rollbackErr != pgx.ErrTxClosed {
				// Логируем ошибку отката, но не переопределяем основную ошибку
				fmt.Errorf("Failed to rollback transaction", zap.Error(rollbackErr))
			} else {
				kp.log.Info("Transaction rolled back due to an error")
			}
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

	// Проверка ошибок выполнения запросов
	for i := 0; i < len(products); i++ {
		if _, execErr := br.Exec(); execErr != nil {
			br.Close() // Закрываем батч перед возвратом ошибки
			err = fmt.Errorf("failed to execute batch query: %w", execErr)
			return err
		}
	}

	// Закрываем батч после обработки всех запросов
	if closeErr := br.Close(); closeErr != nil {
		err = fmt.Errorf("failed to close batch results: %w", closeErr)
		return err
	}

	// Коммит транзакции
	if commitErr := tx.Commit(context.Background()); commitErr != nil {
		err = fmt.Errorf("failed to commit transaction: %w", commitErr)
		return err
	}

	kp.log.Info("Products successfully inserted into the database")
	return nil
}

func (kp *BDKeeper) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	// Проверка подключения к базе
	if kp.pool == nil {
		return nil, fmt.Errorf("database connection pool is nil")
	}

	// SQL-запрос для извлечения всех данных из таблицы
	query := `
		SELECT id, name, category, price, create_date
		FROM prices
	`

	// Выполняем запрос
	rows, err := kp.pool.Query(ctx, query)
	if err != nil {
		kp.log.Info("Failed to execute query", zap.Error(err))
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Считываем данные
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
			kp.log.Info("Failed to scan row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		products = append(products, product)
	}

	// Проверяем наличие ошибок при итерации
	if rows.Err() != nil {
		kp.log.Info("Error occurred during rows iteration", zap.Error(rows.Err()))
		return nil, fmt.Errorf("error during rows iteration: %w", rows.Err())
	}

	kp.log.Info("Successfully retrieved all products", zap.Int("count", len(products)))
	return products, nil
}

func findMigrationsDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("ошибка получения текущего каталога: %w", err)
	}

	for {
		migrationsPath := filepath.Join(currentDir, "migrations")
		info, err := os.Stat(migrationsPath)
		if err == nil && info.IsDir() {
			return migrationsPath, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break // Достигли корневого каталога
		}
		currentDir = parentDir
	}

	return "", fmt.Errorf("папка 'migrations' не найдена")
}
