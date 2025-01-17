package storage

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/drstein77/priceanalyzer/internal/models"
	"go.uber.org/zap"
)

// ErrConflict indicates a data conflict in the store.
var (
	ErrConflict = errors.New("data conflict")
	ErrNotFound = errors.New("not found")
)

type Log interface {
	Info(string, ...zap.Field)
}

// MemoryStorage represents an in-memory storage with locking mechanisms
type MemoryStorage struct {
	ctx context.Context
	mx  sync.RWMutex

	keeper Keeper
	log    Log
}

// Keeper interface for database operations
type Keeper interface {
	GetAllProducts(context.Context) ([]models.Product, error)
	InsertProducts(context.Context, []models.Product) error
	Ping(context.Context) bool
	Close() bool
}

// NewMemoryStorage creates a new MemoryStorage instance
func NewMemoryStorage(ctx context.Context, keeper Keeper, log Log) *MemoryStorage {
	return &MemoryStorage{
		ctx: ctx,

		keeper: keeper,
		log:    log,
	}
}

// GetAllProducts извлекает все продукты через bdKeeper
func (s *MemoryStorage) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	// Вызываем метод GetAllProducts на уровне BDKeeper
	products, err := s.keeper.GetAllProducts(ctx)
	if err != nil {
		return nil, err
	}

	return products, nil
}

func (s *MemoryStorage) ProcessPrices(ctx context.Context, data io.Reader) (*models.ProcessResponse, error) {
	// Чтение CSV-данных
	products, err := s.parseCSV(data)
	if err != nil {
		return nil, err
	}

	// Сохранение данных в базе
	if err := s.keeper.InsertProducts(ctx, products); err != nil {
		return nil, err
	}

	// Сбор статистики
	return s.calculateStats(products), nil
}

func (s *MemoryStorage) parseCSV(data io.Reader) ([]models.Product, error) {
	csvReader := csv.NewReader(bufio.NewReader(data))

	// Пропускаем заголовок CSV
	_, err := csvReader.Read()
	if err != nil {
		return nil, errors.New("failed to read CSV header")
	}

	var products []models.Product
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("failed to read CSV")
		}

		// Преобразуем строку в структуру
		id, _ := strconv.Atoi(record[0])
		price, _ := strconv.ParseFloat(record[3], 64)

		createdAt, err := time.Parse(time.RFC3339, record[4])
		products = append(products, models.Product{
			ID:        id,
			Name:      record[1],
			Category:  record[2],
			Price:     price,
			CreatedAt: createdAt,
		})
	}
	return products, nil
}

func (s *MemoryStorage) calculateStats(products []models.Product) *models.ProcessResponse {
	totalItems := len(products)
	totalPrice := 0.0
	categories := make(map[string]bool)

	for _, product := range products {
		totalPrice += product.Price
		categories[product.Category] = true
	}

	return &models.ProcessResponse{
		TotalItems:      totalItems,
		TotalCategories: len(categories),
		TotalPrice:      totalPrice,
	}
}
