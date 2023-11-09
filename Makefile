docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

create-topic:
	@docker exec -it kafka-ppe kafka-topics.sh --botstrap-server localhost:9092 --create --topic $(name)

delete-topic:
	@docker exec -it kafka-ppe kafka-topics.sh --botstrap-server localhost:9092 --delete --topic $(name)

order-service:
	@go run ./app/services/order-service

inventory-consumer:
	@go run ./app/services/inventory-consumer