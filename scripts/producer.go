package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
)

type TestOrder struct {
	OrderUID          string       `json:"order_uid"`
	TrackNumber       string       `json:"track_number"`
	Entry             string       `json:"entry"`
	Delivery          TestDelivery `json:"delivery"`
	Payment           TestPayment  `json:"payment"`
	Items             []TestItem   `json:"items"`
	Locale            string       `json:"locale"`
	InternalSignature string       `json:"internal_signature"`
	CustomerID        string       `json:"customer_id"`
	DeliveryService   string       `json:"delivery_service"`
	ShardKey          string       `json:"shardkey"`
	SmID              int          `json:"sm_id"`
	DateCreated       time.Time    `json:"date_created"`
	OofShard          string       `json:"oof_shard"`
}

type TestDelivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type TestPayment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDt    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

type TestItem struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

func main() {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	orders := createTestOrders()

	topic := "orders"

	for i, order := range orders {
		orderJSON, err := json.Marshal(order)
		if err != nil {
			log.Printf("Failed to marshal order %d: %v", i, err)
			continue
		}

		message := &sarama.ProducerMessage{
			Topic: topic,
			Key:   sarama.StringEncoder(order.OrderUID),
			Value: sarama.ByteEncoder(orderJSON),
		}

		partition, offset, err := producer.SendMessage(message)
		if err != nil {
			log.Printf("Failed to send message %d: %v", i, err)
			continue
		}

		fmt.Printf("Message %d sent successfully - Topic: %s, Partition: %d, Offset: %d, OrderUID: %s\n",
			i+1, topic, partition, offset, order.OrderUID)

		time.Sleep(1 * time.Second)
	}

	fmt.Println("Тестовые заказы с Order UIDs:")
	for _, order := range orders {
		fmt.Printf("- %s\n", order.OrderUID)
	}
}

func createTestOrders() []TestOrder {
	now := time.Now()

	return []TestOrder{
		{
			OrderUID:    "b563feb7b2b84b6test",
			TrackNumber: "WBILMTESTTRACK",
			Entry:       "WBIL",
			Delivery: TestDelivery{
				Name:    "Test Testov",
				Phone:   "+9720000000",
				Zip:     "2639809",
				City:    "Kiryat Mozkin",
				Address: "Ploshad Mira 15",
				Region:  "Kraiot",
				Email:   "test@gmail.com",
			},
			Payment: TestPayment{
				Transaction:  "b563feb7b2b84b6test",
				RequestID:    "",
				Currency:     "USD",
				Provider:     "wbpay",
				Amount:       1817,
				PaymentDt:    1637907727,
				Bank:         "alpha",
				DeliveryCost: 1500,
				GoodsTotal:   317,
				CustomFee:    0,
			},
			Items: []TestItem{
				{
					ChrtID:      9934930,
					TrackNumber: "WBILMTESTTRACK",
					Price:       453,
					Rid:         "ab4219087a764ae0btest",
					Name:        "Mascaras",
					Sale:        30,
					Size:        "0",
					TotalPrice:  317,
					NmID:        2389212,
					Brand:       "Vivienne Sabo",
					Status:      202,
				},
			},
			Locale:            "en",
			InternalSignature: "",
			CustomerID:        "test",
			DeliveryService:   "meest",
			ShardKey:          "9",
			SmID:              99,
			DateCreated:       now,
			OofShard:          "1",
		},
		{
			OrderUID:    "test_order_12345",
			TrackNumber: "TRACK12345",
			Entry:       "WBIL",
			Delivery: TestDelivery{
				Name:    "Иван Иванов",
				Phone:   "+79999999999",
				Zip:     "101000",
				City:    "Москва",
				Address: "ул. Тверская, д. 1",
				Region:  "Московская область",
				Email:   "ivan@example.com",
			},
			Payment: TestPayment{
				Transaction:  "test_transaction_12345",
				RequestID:    "req_12345",
				Currency:     "RUB",
				Provider:     "sberbank",
				Amount:       5500,
				PaymentDt:    now.Unix(),
				Bank:         "sberbank",
				DeliveryCost: 500,
				GoodsTotal:   5000,
				CustomFee:    0,
			},
			Items: []TestItem{
				{
					ChrtID:      12345,
					TrackNumber: "TRACK12345",
					Price:       2500,
					Rid:         "rid12345",
					Name:        "Смартфон IPhone 16MAX",
					Sale:        0,
					Size:        "128GB",
					TotalPrice:  2500,
					NmID:        123456,
					Brand:       "Apple",
					Status:      200,
				},
			},
			Locale:            "ru",
			InternalSignature: "signature",
			CustomerID:        "customer_001",
			DeliveryService:   "cdek",
			ShardKey:          "1",
			SmID:              1,
			DateCreated:       now,
			OofShard:          "2",
		},
	}
}
