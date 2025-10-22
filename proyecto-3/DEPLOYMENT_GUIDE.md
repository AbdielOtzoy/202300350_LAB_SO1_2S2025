# Quick Deployment Guide - All in Default Namespace

## Prerequisites
- GKE cluster created with the command in `yaml/cluster.txt`
- `kubectl` configured to access your cluster
- Strimzi Operator for Kafka

## Simple Deployment (Copy & Paste)

```bash
# 1. Install Strimzi Operator for Kafka
kubectl create -f 'https://strimzi.io/install/latest?namespace=default'

# 2. Deploy everything in order
kubectl apply -f yaml/valkey-deployment.yaml
kubectl apply -f yaml/brokers/
kubectl apply -f yaml/grcp/
kubectl apply -f yaml/rust-api/
kubectl apply -f yaml/consumers/
kubectl apply -f yaml/grafana-deplyment.yaml

# 3. Wait for everything to be ready (this may take 3-5 minutes)
kubectl get pods -w
```

## Service Endpoints (All in Default Namespace)

| Service | Internal DNS | Port | Purpose |
|---------|-------------|------|---------|
| Kafka | `my-cluster-kafka-bootstrap` | 9092 | Message Broker |
| RabbitMQ | `rabbitmq` | 5672, 15672 | Message Broker + UI |
| Valkey | `valkey-service` | 6379 | Data Storage |
| Rust API | `rust-api-service` | 8080 | REST Endpoint |
| Receptor | `receptor-service` | 50051 | gRPC Router |
| Publisher Rabbit | `publisher-rabbit-service` | 50052 | RabbitMQ Publisher |
| Publisher Kafka | `publisher-kafka-service` | 50053 | Kafka Publisher |
| Grafana | NodePort | 30300 | Monitoring Dashboard |

## Testing the System

### 1. Get Ingress External IP
```bash
kubectl get ingress rust-api-ingress
# Wait for EXTERNAL-IP to be assigned
```

### 2. Send Test Request
```bash
curl -X POST http://<EXTERNAL-IP>/clima \
  -H "Content-Type: application/json" \
  -d '{"municipality":1,"temperature":25,"humidity":70,"weather":1}'
```

### 3. Verify Data in Valkey
```bash
kubectl exec -it deployment/valkey -- valkey-cli
```
Inside the CLI:
```
KEYS *
HGETALL municipality:mixco
```

### 4. Access Grafana
```bash
# Get node IP
kubectl get nodes -o wide

# Access Grafana at: http://<NODE_IP>:30300
# Username: admin
# Password: admin
```

## Troubleshooting

### Check Pod Status
```bash
kubectl get pods
kubectl describe pod <pod-name>
```

### View Logs
```bash
# Rust API
kubectl logs -f deployment/rust-api --tail=50

# Receptor (router)
kubectl logs -f deployment/receptor --tail=50

# Publishers
kubectl logs -f deployment/publisher-rabbit --tail=50
kubectl logs -f deployment/publisher-kafka --tail=50

# Consumers
kubectl logs -f deployment/consumer-rabbit --tail=50
kubectl logs -f deployment/kafka-consumer --tail=50

# RabbitMQ
kubectl logs -f deployment/rabbitmq --tail=50

# Kafka
kubectl logs -f my-cluster-kafka-0 --tail=50
```

### Check Service Connectivity
```bash
# Test from inside the cluster
kubectl run test-pod --rm -it --image=curlimages/curl -- sh

# Inside the pod:
curl http://rust-api-service:8080/
curl http://rabbitmq:15672
```

### Resource Usage
```bash
kubectl top nodes
kubectl top pods
```

## Data Flow Verification

1. **Send Request** → Rust API receives JSON at `/clima`
2. **gRPC Call** → Rust API forwards to Receptor (50051)
3. **Random Route** → Receptor randomly selects:
   - RabbitMQ Publisher (50052) → RabbitMQ → Consumer → Valkey
   - Kafka Publisher (50053) → Kafka → Consumer → Valkey
4. **Data Storage** → Valkey stores with 1-hour TTL
5. **Visualization** → Grafana queries Valkey data

## Clean Up

```bash
# Delete all resources
kubectl delete -f yaml/consumers/
kubectl delete -f yaml/rust-api/
kubectl delete -f yaml/grcp/
kubectl delete -f yaml/brokers/
kubectl delete -f yaml/valkey-deployment.yaml
kubectl delete -f yaml/grafana-deplyment.yaml

# Delete Strimzi operator
kubectl delete -f 'https://strimzi.io/install/latest?namespace=default'
```
