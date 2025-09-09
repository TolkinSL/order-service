package cache

import (
	"order-service/internal/models"
	"sync"

	"github.com/sirupsen/logrus"
)

type MemoryCache struct {
	orders map[string]*models.Order
	mutex  sync.RWMutex
	log    *logrus.Logger
}

func NewMemoryCache(logger *logrus.Logger) *MemoryCache {
	return &MemoryCache{
		orders: make(map[string]*models.Order),
		log:    logger,
	}
}

func (c *MemoryCache) Set(orderUID string, order *models.Order) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.orders[orderUID] = order
	c.log.Debugf("Order %s added to cache", orderUID)
}

func (c *MemoryCache) Get(orderUID string) (*models.Order, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	order, exists := c.orders[orderUID]
	if exists {
		c.log.Debugf("Order %s found in cache", orderUID)
		return order, true
	}

	c.log.Debugf("Order %s not found in cache", orderUID)
	return nil, false
}

func (c *MemoryCache) Delete(orderUID string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.orders, orderUID)
	c.log.Debugf("Order %s deleted from cache", orderUID)
}

func (c *MemoryCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.orders = make(map[string]*models.Order)
	c.log.Debug("Cache cleared")
}

func (c *MemoryCache) LoadFromRepository(repo OrderRepository) error {
	c.log.Info("Loading orders from repository to cache...")

	orders, err := repo.GetAllOrders()
	if err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	for _, order := range orders {
		c.orders[order.OrderUID] = order
	}

	c.log.Infof("Loaded %d orders into cache", len(orders))
	return nil
}

func (c *MemoryCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.orders)
}

func (c *MemoryCache) GetAll() map[string]*models.Order {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	result := make(map[string]*models.Order)
	for k, v := range c.orders {
		result[k] = v
	}

	return result
}

type OrderRepository interface {
	GetAllOrders() ([]*models.Order, error)
	GetOrder(orderUID string) (*models.Order, error)
}
