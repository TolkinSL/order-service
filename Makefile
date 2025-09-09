.PHONY: setup run producer stop

APP_NAME=order-service
DOCKER_COMPOSE=docker-compose

# Запуск Docker (Postgres + Kafka)
setup:
	@echo "Запуск Docker сервисов"
	$(DOCKER_COMPOSE) up -d
	@echo "-Пауза для запуска сервисов"
	sleep 10
	@echo "-Применение миграции"
	docker exec -i orders_postgres psql -U orders_user -d orders_db < migrations/001_create_tables.sql
	@echo "-Всё готово!"

# Сборка и запуск приложения
run:
	@echo "-Сборка приложения"
	go build -o bin/$(APP_NAME) cmd/main.go
	@echo "-Запуск приложения"
	@exec ./bin/$(APP_NAME)

# Сборка и запуск продьюсера
producer:
	@echo "-Сборка producer"
	go build -o bin/producer scripts/producer.go
	@echo "-Отправка тестовых заказов в Kafka"
	@exec ./bin/producer

# Остановить и удалить Docker сервисы
stop:
	@echo "-Остановка Docker сервисов"
	$(DOCKER_COMPOSE) down -v
	@echo "-Все сервисы остановлены и данные удалены!"
	@echo "-Очистка"
	rm -rf bin/
	go clean -cache
