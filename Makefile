DOCKER_COMPOSE = docker compose

PROTO_DIR = api
OUT_DIR = internal/transport/grpc/pb
MODULE_NAME = github.com/wildberries-test-task-22-05-2026

PROTO_FILES = $(shell find $(PROTO_DIR) -name "*.proto")

.DEFAULT_GOAL := help
.PHONY: help run stop test bench proto lint clean tidy seed

help:
	@echo "Доступные команды:"
	@echo "	make run	- запуск проекта и инфраструктуры в Docker"
	@echo "	make stop	- Остановка всех контейнеров и удаление volume-ов"
	@echo "	make test	- Запуск всех unit-тестов"
	@echo "	make bench	- Запуск бенчмарков для ядра"
	@echo "	make proto	- Генерация gRPC кода из .proto файлов"
	@echo "	make lint	- Прогон линтера"
	@echo "	make tidy	- Обновление и загрузка Go-зависимостей"
	@echo "	load-test	- E2E нагрузочное тестирование через ghz"

run:
	$(DOCKER_COMPOSE) up -d --build

stop:
	$(DOCKER_COMPOSE) down -v

test:
	go test -v -race ./...

bench:
	go test -bench=. -benchmem -cpu=4 ./internal/repository/memory/...

proto:
	@mkdir -p $(OUT_DIR)
	protoc -I $(PROTO_DIR) \
		--go_out=. --go_opt=module=$(MODULE_NAME) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE_NAME) \
		$(PROTO_FILES)

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

seed:
	@echo "Наполняем Kafka тестовыми данными"
	go run cmd/seeder/main.go

load-test: seed
	@echo "Ждем 5 секунды, пока консьюмер обработает очередь"
	@sleep 5
	@echo "Тест через ghz"
	ghz --insecure \
		--proto=api/trend/v1/trend.proto \
		--call=trendservice.TrendService.GetTopN \
		--data='{"limit": 10}' \
		-n 1000000 \
		-c 128 \
		127.0.0.1:50051
