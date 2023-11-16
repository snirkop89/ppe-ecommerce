docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

create-topic:
	@docker exec -it kafka-ppe kafka-topics.sh --botstrap-server localhost:9092 --create --topic $(name)

delete-topic:
	@docker exec -it kafka-ppe kafka-topics.sh --botstrap-server localhost:9092 --delete --topic $(name)

## Services
SERVICES =order-service inventory-consumer notification shipper warehouse 
.PHONY = $(SERVICES)

run-all: $(SERVICES)

stop-all:
	killall $(SERVICES)
	go clean

build/order-service:
	@go build -o bin/order-service ./app/services/order-service

order-service: build/order-service
	@./bin/order-service -addr ':8000' &

build/inventory-consumer:
	@go build -o bin/inventory-consumer ./app/services/inventory-consumer

inventory-consumer: build/inventory-consumer
	@./bin/inventory-consumer -addr ':8001' &

build/notification:
	@go build -o bin/notification ./app/services/notification

notification: build/notification
	@./bin/notification -addr ':8002' &

build/shipper:
	@go build -o bin/shipper ./app/services/shipper

shipper: build/shipper
	@./bin/shipper -addr ':8003' &

build/warehouse:
	@go build -o bin/warehouse ./app/services/warehouse

warehouse: build/warehouse
	@./bin/warehouse -addr ':8004' &