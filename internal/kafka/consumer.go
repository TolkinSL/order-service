package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"order-service/internal/models"
	"sync"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"
)

type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	topics        []string
	handlers      []MessageHandler
	log           *logrus.Logger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

type MessageHandler interface {
	HandleOrder(order *models.Order) error
}

func NewConsumer(brokers []string, groupID string, topics []string, logger *logrus.Logger) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Group.Session.Timeout = 10000000000
	config.Consumer.Group.Heartbeat.Interval = 3000000000

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		consumerGroup: consumerGroup,
		topics:        topics,
		log:           logger,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

func (c *Consumer) AddHandler(handler MessageHandler) {
	c.handlers = append(c.handlers, handler)
}

func (c *Consumer) Start() error {
	c.log.Info("Starting Kafka consumer...")

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				c.log.Info("Consumer context cancelled")
				return
			default:
				if err := c.consumerGroup.Consume(c.ctx, c.topics, c); err != nil {
					c.log.Errorf("Error from consumer: %v", err)
				}
			}
		}
	}()

	go func() {
		for err := range c.consumerGroup.Errors() {
			c.log.Errorf("Consumer error: %v", err)
		}
	}()

	c.log.Info("Kafka consumer started successfully")
	return nil
}

func (c *Consumer) Stop() error {
	c.log.Info("Stopping Kafka consumer...")
	c.cancel()
	c.wg.Wait()
	return c.consumerGroup.Close()
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	c.log.Info("Consumer group session setup")
	return nil
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	c.log.Info("Consumer group session cleanup")
	return nil
}

func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			c.log.Debugf("Received message from topic %s, partition %d, offset %d",
				message.Topic, message.Partition, message.Offset)

			if err := c.processMessage(message); err != nil {
				c.log.Errorf("Failed to process message: %v", err)
			} else {
				session.MarkMessage(message, "")
			}

		case <-session.Context().Done():
			return nil
		}
	}
}

func (c *Consumer) processMessage(message *sarama.ConsumerMessage) error {
	var order models.Order
	if err := json.Unmarshal(message.Value, &order); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	if err := order.Validate(); err != nil {
		c.log.Warnf("Invalid order data: %v", err)
		return err
	}

	c.log.Infof("Processing order: %s", order.OrderUID)

	for _, handler := range c.handlers {
		if err := handler.HandleOrder(&order); err != nil {
			return fmt.Errorf("handler failed to process order: %w", err)
		}
	}

	c.log.Infof("Order %s processed successfully", order.OrderUID)
	return nil
}

type OrderHandler struct {
	repository OrderRepository
	cache      OrderCache
	log        *logrus.Logger
}

type OrderRepository interface {
	SaveOrder(order *models.Order) error
	GetOrder(orderUID string) (*models.Order, error)
}

type OrderCache interface {
	Set(orderUID string, order *models.Order)
	Get(orderUID string) (*models.Order, bool)
}

func NewOrderHandler(repo OrderRepository, cache OrderCache, logger *logrus.Logger) *OrderHandler {
	return &OrderHandler{
		repository: repo,
		cache:      cache,
		log:        logger,
	}
}

func (h *OrderHandler) HandleOrder(order *models.Order) error {
	if err := h.repository.SaveOrder(order); err != nil {
		h.log.Errorf("Failed to save order to database: %v", err)
		return err
	}

	h.cache.Set(order.OrderUID, order)

	h.log.Infof("Order %s handled successfully", order.OrderUID)
	return nil
}
