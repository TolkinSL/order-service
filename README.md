## Order Service

Демонстрационный сервис для обработки заказов с использованием Kafka, PostgreSQL и кеширования в памяти.

Сервис получает данные о заказах из очереди сообщений Kafka, сохраняет их в PostgreSQL, кэширует в памяти для быстрого доступа и предоставляет HTTP API для получения информации о заказах. Также включает веб-интерфейс для удобного просмотра заказов.

Ссылка на Видео работы сервиса - https://disk.yandex.ru/d/v2JFmNpw__TnQw

### 1. Сборка и запуск сервиса

```bash
# Сборка приложения
make setup

# Запуск сервиса
make run
```

### 2. Отправка тестовых данных

В другом терминале:

```bash
# Отправка тестовых заказов в Kafka
make producer
```

### 3. Проверка работы

- **Веб-интерфейс**: http://localhost:8081
- **API**: http://localhost:8081/api/v1/order/b563feb7b2b84b6test
- **API**: http://localhost:8081/api/v1/order/test_order_12345
- **Health Check**: http://localhost:8081/api/v1/health
- **Cache Stats**: http://localhost:8081/api/v1/cache/stats

### 4. Остановка сервиса

```bash
# Остановка сервиса
make stop
```

## API Endpoints

### Получить заказ по ID
```http
GET /api/v1/order/{order_uid}
```

### Health Check
```http
GET /api/v1/health
```

### Статистика кеша
```http
GET /api/v1/cache/stats
```

## Примеры использования

### Отправка заказа в Kafka (curl)

```bash
# Отправить JSON из файла order.json
docker exec -i orders_kafka \
  kafka-console-producer --topic orders --bootstrap-server localhost:9092 < order.json

```

### Переменные окружения:
   ```bash
   export DB_HOST=your-postgres-host
   export DB_PASSWORD=secure-password
   export KAFKA_BROKERS=your-kafka-brokers
   export SERVER_PORT=8081
   ```
