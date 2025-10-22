# GKE Resource Optimization Report

## Cluster Configuration
- **Machine Type:** e2-medium (2 vCPUs, 4GB RAM per node)
- **Nodes:** 3 (min: 2, max: 5)
- **Total Capacity:** 6 vCPUs, 12GB RAM
- **Available for Pods:** ~5.2 vCPUs, ~10GB RAM (after system overhead)

## Issues Fixed

### 🚨 Critical: Service Name Corrections
**All services are in `default` namespace for simplified networking**

**Updated Service References (all in default namespace):**
- Kafka: `my-cluster-kafka-bootstrap:9092`
- RabbitMQ: `rabbitmq:5672`
- Valkey: `valkey-service:6379`
- Rust API: `rust-api-service:8080`
- Receptor: `receptor-service:50051`
- Publishers: `publisher-rabbit-service:50052`, `publisher-kafka-service:50053`

### ⚡ Resource Optimization

#### BEFORE Optimization:
```
Total Requests: 1650m CPU (1.65 cores), 3136Mi RAM (~3.1GB)
Total Limits:   4100m CPU (4.1 cores), 6144Mi RAM (~6GB)
```

#### AFTER Optimization:
```
Total Requests: 1020m CPU (1.02 cores), 1828Mi RAM (~1.8GB)
Total Limits:   2360m CPU (2.36 cores), 3540Mi RAM (~3.5GB)
```

**Savings:** 38% CPU requests, 42% RAM requests

### Specific Changes:

| Component | CPU Req | RAM Req | CPU Limit | RAM Limit |
|-----------|---------|---------|-----------|-----------|
| **Kafka** | 150m (was 200m) | 384Mi (was 512Mi) | 400m (was 500m) | 768Mi (was 1Gi) |
| **Kafka Operators** | 60m (was 100m) | 192Mi (was 256Mi) | 160m (was 200m) | 384Mi (was 512Mi) |
| **RabbitMQ** | 100m (was 200m) | 192Mi (was 256Mi) | 300m (was 500m) | 384Mi (was 512Mi) |
| **Valkey** | 100m (was 400m) | 256Mi (was 1Gi) | 300m (was 1Gi) | 512Mi (was 1.5Gi) |
| **Rust API** | 100m | 128Mi | 500m | 512Mi |
| **Grafana** | 50m (was 100m) | 128Mi (was 256Mi) | 200m (was 500m) | 256Mi (was 512Mi) |
| **Go Services** | 100m each | 128Mi each | 500m each | 512Mi each |
| **Consumers** | 50-100m | 64-128Mi | 200-500m | 256-512Mi |

## Resource Distribution (all in default namespace)

### Message Brokers:
- Kafka: 150m CPU, 384Mi RAM
- Kafka Operators: 60m CPU, 192Mi RAM
- RabbitMQ: 100m CPU, 192Mi RAM
- **Subtotal:** 310m CPU, 768Mi RAM

### Application Services:
- Rust API: 100m CPU, 128Mi RAM
- Receptor: 100m CPU, 128Mi RAM
- Publisher Rabbit: 100m CPU, 128Mi RAM
- Publisher Kafka: 100m CPU, 128Mi RAM
- Consumer Rabbit: 100m CPU, 128Mi RAM
- Consumer Kafka: 50m CPU, 64Mi RAM
- **Subtotal:** 550m CPU, 712Mi RAM

### Storage & Monitoring:
- Valkey: 100m CPU, 256Mi RAM
- Grafana: 50m CPU, 128Mi RAM
- **Subtotal:** 150m CPU, 384Mi RAM

### TOTAL: 1010m CPU (~1 core), 1864Mi RAM (~1.8GB)

## Deployment Order

```bash
# 1. Install Strimzi Operator (for Kafka) in default namespace
kubectl create -f 'https://strimzi.io/install/latest?namespace=default'

# 2. Deploy storage layer
kubectl apply -f yaml/valkey-deployment.yaml

# 3. Deploy message brokers
kubectl apply -f yaml/brokers/

# 4. Wait for brokers to be ready
kubectl wait --for=condition=Ready kafka/my-cluster --timeout=300s
kubectl wait --for=condition=Ready pod -l app=rabbitmq --timeout=120s

# 5. Deploy application services (receptor and publishers)
kubectl apply -f yaml/grcp/

# 6. Deploy Rust API
kubectl apply -f yaml/rust-api/

# 7. Deploy consumers
kubectl apply -f yaml/consumers/

# 8. Deploy monitoring
kubectl apply -f yaml/grafana-deplyment.yaml

# 9. Verify all pods are running
kubectl get pods -w
```

## Verification Commands

```bash
# Check resource usage
kubectl top nodes
kubectl top pods

# Verify all pods are running
kubectl get pods
kubectl get svc

# Check specific components
kubectl logs -l app=rust-api
kubectl logs -l app=receptor
kubectl logs -l app=publisher-rabbit
kubectl logs -l app=publisher-kafka
kubectl logs -l app=consumer-rabbit
kubectl logs -l app=kafka-consumer

# Test the full data flow
curl -X POST http://<NODE_IP>:<INGRESS_PORT>/clima \
  -H "Content-Type: application/json" \
  -d '{"municipality":1,"temperature":25,"humidity":70,"weather":1}'

# Check data in Valkey
kubectl exec -it deployment/valkey -- valkey-cli
# Inside valkey-cli:
KEYS *
HGETALL municipality:mixco
```

## Recommendations

1. **Monitor Resource Usage:** Use `kubectl top` regularly to see actual consumption
2. **Enable HPA:** For Rust API if traffic increases:
   ```yaml
   apiVersion: autoscaling/v2
   kind: HorizontalPodAutoscaler
   metadata:
     name: rust-api-hpa
   spec:
     scaleTargetRef:
       apiVersion: apps/v1
       kind: Deployment
       name: rust-api
     minReplicas: 1
     maxReplicas: 3
     metrics:
     - type: Resource
       resource:
         name: cpu
         target:
           type: Utilization
           averageUtilization: 70
   ```

3. **Cost Optimization:** Current setup can run on 2 nodes during low traffic
4. **Storage:** Add PersistentVolumeClaims for Kafka and Valkey in production
5. **Monitoring:** Grafana is accessible at `http://<NODE_IP>:30300`

## Estimated Costs (GKE us-central1-a)
- e2-medium: ~$0.0334/hour
- 3 nodes: ~$0.10/hour = ~$72/month
- With autoscaling to 2 nodes: ~$48/month
- Persistent disks (if added): ~$20/month for 50GB
