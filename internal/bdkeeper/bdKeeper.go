package bdkeeper

import (
	"context"
	"fmt"
	"os"
	"time"

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

func NewBDKeeper(dsn func() string, log Log, userUpdateInterval func() string) *BDKeeper {
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
		pool:               pool,
		log:                log,
		userUpdateInterval: userUpdateInterval,
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
