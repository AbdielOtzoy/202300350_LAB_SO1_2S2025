# WeatherTweet Microservices - AI Agent Instructions

## Architecture Overview

This is a distributed weather data collection system with dual-broker architecture (RabbitMQ + Kafka) deployed on Kubernetes.

**Data Flow:**
1. **Rust API** (`src/api_rust/`) - REST endpoint at `:8080/clima` receiving weather JSON
2. **Go Receptor** (`src/go_service/receptor/`) - gRPC server at `:50051` that randomly routes to either RabbitMQ or Kafka publisher
3. **Go Publishers** - Two separate gRPC services:
   - **RabbitMQ Publisher** (`:50052`) - Publishes to topic exchange `weather_exchange` with routing key `weather.tweet.{municipality}.{weather}`
   - **Kafka Publisher** (`:50053`) - Publishes to topic `weather_tweets` with municipality as partition key
4. **Consumers** - Store data in Valkey (Redis fork):
   - RabbitMQ consumer reads from `weather_tweets_queue`
   - Kafka consumer reads from `weather_tweets` topic with consumer group `weather_consumers`

## Project Structure

- `src/api_rust/` - Actix-web REST API with tonic gRPC client
- `src/go_service/` - All Go services sharing the same protobuf definitions
  - `receptor/` - Main entry point gRPC server
  - `publisher/` - RabbitMQ publisher
  - `kafka/` - Kafka publisher
  - `consumer/` - RabbitMQ consumer
  - `consumer_kafka/` - Kafka consumer
  - `proto/` - Generated protobuf code (shared across all Go services)
- `yaml/` - Kubernetes manifests organized by component type

## Critical Patterns

### 1. Protobuf Schema (`src/*/proto/weathertweet.proto`)
Weather data uses **enums** for municipalities and weather types. Always convert enum values to strings before publishing to message brokers:
```go
municipality := getMunicipalityName(req.Municipality) // "mixco", "guatemala", etc.
weather := getWeatherName(req.Weather)               // "sunny", "cloudy", etc.
```

### 2. Multi-Service Go Project
All Go services **share the same `go.mod`** but use different Dockerfiles targeting specific subdirectories:
- `Dockerfile.receptor` builds `./receptor/receptor.go`
- `Dockerfile.rabbit` builds `./publisher/publisher_rabbit.go`
- `Dockerfile.kafka` builds `./kafka/publisher_kafka.go`

**Important:** When adding dependencies, run `go mod tidy` once at the root `src/go_service/` level.

### 3. Message Broker Patterns

**RabbitMQ:**
- Exchange: `weather_exchange` (type: topic, durable)
- Queue: `weather_tweets_queue` (durable, bound with `weather.tweet.#`)
- Routing key format: `weather.tweet.{municipality}.{weather}`
- Connection string from env: `RABBITMQ_URL` (format: `amqp://admin:admin123@rabbitmq.brokers.svc.cluster.local:5672/`)

**Kafka:**
- Topic: `weather_tweets` (auto-created by Strimzi operator)
- Partition key: municipality name (ensures same municipality goes to same partition)
- Broker address from env: `KAFKA_BROKER_ADDRESS` (format: `kafka-service:9092`)
- Uses Sarama library with `RequiredAcks = WaitForAll` for reliability

### 4. Kubernetes Deployment Strategy

**Namespace organization:**
- `brokers` - Kafka (Strimzi operator) and RabbitMQ deployments
- `default` - Application services (Rust API, Go publishers, consumers)

**Service discovery:**
- Cross-namespace access: `{service}.{namespace}.svc.cluster.local`
- Example: `rabbitmq.brokers.svc.cluster.local:5672`

**Critical environment variables:**
- Rust API: `GRPC_SERVER_URL` points to receptor service
- Receptor: `RABBITMQ_PUBLISHER_ADDRESS`, `KAFKA_PUBLISHER_ADDRESS`, `RECEIVER_PORT`
- Publishers: `RABBITMQ_URL` or `KAFKA_BROKER_ADDRESS`
- Consumers: Broker address + `VALKEY_ADDRESS` or `VALKEY_SERVICE_URL`

### 5. Data Storage (Valkey/Redis)
RabbitMQ consumer stores as hash: `municipality:{name}` with fields `name`, `temperature`, `humidity`, `weather`, `last_update`

Kafka consumer stores as: `weather:{municipality}:{timestamp}` with full JSON

Keys expire after 1 hour.

## Development Workflows

### Building Locally
**Rust API:**
```bash
cd src/api_rust
cargo build --release
# Protobuf compilation handled by build.rs
```

**Go Services:**
```bash
cd src/go_service
# Regenerate protobuf (if .proto changed)
protoc --go_out=. --go-grpc_out=. proto/weathertweet.proto
# Build specific service
go build -o receptor ./receptor/receptor.go
```

### Docker Build Pattern
All services use multi-stage builds (alpine-based). Images are pushed to registry: `ecb46a7a9745.ngrok-free.app/{service-name}:latest`

### Kubernetes Deployment
```bash
# Deploy brokers first
kubectl apply -f yaml/brokers/
# Wait for brokers to be ready
kubectl apply -f yaml/grcp/    # Publishers & receptor
kubectl apply -f yaml/rust-api/ # API + ingress
kubectl apply -f yaml/consumers/ # Consumers
```

## Testing the System
Send POST to ingress:
```bash
curl -X POST http://{ingress-ip}/clima \
  -H "Content-Type: application/json" \
  -d '{"municipality":1,"temperature":25,"humidity":70,"weather":1}'
```

Check logs:
```bash
kubectl logs -f deployment/publisher-rabbit  # See RabbitMQ path
kubectl logs -f deployment/publisher-kafka   # See Kafka path
kubectl logs -f deployment/consumer-rabbit   # Verify storage
```

## Common Gotchas

1. **Protobuf sync:** Rust and Go proto files must match. Rust uses `tonic-build` in `build.rs`, Go uses `protoc` with `--go-grpc_out`
2. **gRPC service names:** Receptor implements `WeatherTweetService`, publishers implement `WeatherPublisherService` (different proto services)
3. **Random routing:** Receptor uses `rand.Intn(2)` to balance between brokers - not based on load or availability
4. **Valkey vs Redis:** Uses `valkey-go` client (not `go-redis`) for Kafka consumer, but RabbitMQ consumer uses `go-redis`
5. **Strimzi Kafka:** Uses Kafka CRDs (`KafkaNodePool`, `Kafka`), not standard StatefulSets. Topic creation is automatic.
