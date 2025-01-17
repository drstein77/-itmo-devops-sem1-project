package storage

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
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

// Log defines an interface for logging.
type Log interface {
	Info(string, ...zap.Field)
	Error(string, ...zap.Field)
}

// MemoryStorage represents an in-memory storage with locking mechanisms.
type MemoryStorage struct {
	ctx context.Context
	mx  sync.RWMutex

	keeper Keeper
	log    Log
}

// Keeper is an interface for database operations.
type Keeper interface {
	GetAllProducts(context.Context) ([]models.Product, error)
	InsertProducts(context.Context, []models.Product) error
	Ping(context.Context) bool
	Close() bool
}

// NewMemoryStorage creates a new MemoryStorage instance.
func NewMemoryStorage(ctx context.Context, keeper Keeper, log Log) *MemoryStorage {
	if keeper == nil {
		log.Error("keeper is nil, cannot initialize storage")
		return nil
	}

	return &MemoryStorage{
		ctx: ctx,

		keeper: keeper,
		log:    log,
	}
}

// GetAllProducts retrieves all products via dbKeeper.
func (s *MemoryStorage) GetAllProducts(ctx context.Context) ([]models.Product, error) {
	s.mx.RLock()
	defer s.mx.RUnlock()

	// Call the GetAllProducts method at the DBKeeper level
	products, err := s.keeper.GetAllProducts(ctx)
	if err != nil {
		return nil, err
	}
	return products, nil
}

func (s *MemoryStorage) ProcessPrices(ctx context.Context, data io.Reader) (*models.ProcessResponse, error) {
	// Read CSV data
	products, err := s.parseCSV(data)
	if err != nil {
		return nil, err
	}

	s.mx.Lock()
	defer s.mx.Unlock()

	// Save data to the database
	if err := s.keeper.InsertProducts(ctx, products); err != nil {
		return nil, err
	}

	// Collect statistics
	return s.calculateStats(products), nil
}

func (s *MemoryStorage) parseCSV(data io.Reader) ([]models.Product, error) {
	csvReader := csv.NewReader(bufio.NewReader(data))

	// Skip the CSV header
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

		// Check if record has the expected number of fields
		if len(record) != 5 {
			return nil, fmt.Errorf("unexpected number of fields in record: %v", record)
		}

		// Parse ID
		id, parseErr := strconv.Atoi(record[0])
		if parseErr != nil {
			return nil, fmt.Errorf("invalid ID format: %v", parseErr)
		}

		// Parse Price
		price, parseErr := strconv.ParseFloat(record[3], 64)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid price format: %v", parseErr)
		}

		// Parse CreatedAt
		createdAt, dateErr := time.Parse("2006-01-02", record[4])
		if dateErr != nil {
			return nil, fmt.Errorf("invalid date format: %v", dateErr)
		}

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
