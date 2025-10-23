# Guía de Despliegue Paso a Paso - WeatherTweet Microservicios

## Requisitos Previos

1. Cluster GKE creado y activo
2. `kubectl` configurado y conectado al cluster
3. Permisos de administrador en el cluster

## Orden de Despliegue

### PASO 1: Instalar el Operador Strimzi (para Kafka)

```bash
kubectl create -f 'https://strimzi.io/install/latest?namespace=default'
```

**Verificación:**
```bash
# Ver que los pods del operador estén corriendo
kubectl get pods -l name=strimzi-cluster-operator

# Esperar a que esté en estado Running
kubectl wait --for=condition=Ready pod -l name=strimzi-cluster-operator --timeout=300s
```

**Salida esperada:** Pod `strimzi-cluster-operator-xxxxx` en estado `Running`

---

### PASO 2: Desplegar Valkey (Almacenamiento de Datos)

```bash
kubectl apply -f yaml/valkey-deployment.yaml
```

**Verificación:**
```bash
# Ver el estado del deployment
kubectl get deployment valkey

# Ver el pod de Valkey
kubectl get pods -l app=valkey

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=valkey --timeout=120s

# Verificar que el servicio esté activo
kubectl get svc valkey-service
```

**Prueba de conectividad:**
```bash
# Probar conexión a Valkey
kubectl run test-valkey-cli --rm -it --image=redis:alpine -- /bin/sh

# Dentro del pod:
redis-cli -h valkey-service -p 6379 PING

# Debe responder: PONG
```

**Salida esperada:** 
- Deployment: `1/1 replicas` listas
- Service: `valkey-service` con ClusterIP asignado

---

### PASO 3: Desplegar Message Brokers (Kafka y RabbitMQ)

#### 3a. Desplegar Kafka

```bash
kubectl apply -f yaml/brokers/kafka-cluster.yaml
```

**Verificación:**
```bash
# Ver el recurso Kafka
kubectl get kafka my-cluster

# Ver los pods de Kafka (puede tardar 3-5 minutos)
kubectl get pods -l strimzi.io/cluster=my-cluster

# Esperar a que Kafka esté listo
kubectl wait --for=condition=Ready kafka/my-cluster --timeout=600s

# Ver los servicios de Kafka
kubectl get svc -l strimzi.io/cluster=my-cluster
```

**Prueba de conectividad:**
```bash
# Crear un pod de prueba para Kafka
kubectl run kafka-test --rm -it --image=quay.io/strimzi/kafka:latest-kafka-3.8.0 -- /bin/bash
# Dentro del pod, listar topics:
bin/kafka-topics.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --list

# Salir: exit
```

**Crear un topic de prueba:**
```bash
kubectl run kafka-test-cli --rm -it --image=quay.io/strimzi/kafka:latest-kafka-3.8.0 -- /bin/bash
# Dentro del Pod 'kafka-test-cli'
bin/kafka-topics.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --create --topic weather_tweets --partitions 1 --replication-factor 1

# Dentro del Pod 'kafka-test-cli'
bin/kafka-topics.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 --list

```


**Salida esperada:**
- Kafka resource: `Ready: True`
- Pods: `my-cluster-kafka-0`, `my-cluster-entity-operator-xxxxx` en Running
- Service: `my-cluster-kafka-bootstrap` activo

- URL dentro de la red del cluster: `my-cluster-kafka-bootstrap:9092`

---

#### 3b. Desplegar RabbitMQ

```bash
kubectl apply -f yaml/brokers/rabbit-deployment.yaml
```

**Verificación:**
```bash
# Ver el deployment
kubectl get deployment rabbitmq

# Ver el pod de RabbitMQ
kubectl get pods -l app=rabbitmq

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=rabbitmq --timeout=120s

# Ver el servicio
kubectl get svc rabbitmq
```

**crear exchange y queue de prueba:**
```bash
RABBIT_POD=$(kubectl get pods -l app=rabbitmq -o jsonpath='{.items[0].metadata.name}')

kubectl exec -it $RABBIT_POD -- rabbitmqadmin declare exchange \
  name=weather_exchange \
  type=topic \
  durable=true \
  --host=127.0.0.1 \
  --port=15672 \
  --vhost=/ \
  --username=admin \
  --password=admin123

kubectl exec -it $RABBIT_POD -- rabbitmqadmin declare queue \
  name=weather_tweets_queue \
  durable=true \
  --host=127.0.0.1 \
  --port=15672 \
  --vhost=/ \
  --username=admin \
  --password=admin123

kubectl exec -it $RABBIT_POD -- rabbitmqadmin declare binding \
  source=weather_exchange \
  destination=weather_tweets_queue \
  routing_key="weather.tweet.#" \
  --host=127.0.0.1 \
  --port=15672 \
  --vhost=/ \
  --username=admin \
  --password=admin123

# 1. Listar Exchanges
kubectl exec -it $RABBIT_POD -- rabbitmqadmin list exchanges \
  --host=127.0.0.1 --port=15672 \
  --username=admin --password=admin123 \
  name type

# 2. Listar Queues
kubectl exec -it $RABBIT_POD -- rabbitmqadmin list queues \
  --host=127.0.0.1 --port=15672 \
  --username=admin --password=admin123 \
  name messages

# 3. Listar Bindings
kubectl exec -it $RABBIT_POD -- rabbitmqadmin list bindings \
  --host=127.0.0.1 --port=15672 \
  --username=admin --password=admin123 \
  source destination routing_key
```

**Prueba de conectividad:**
```bash
# Verificar logs de RabbitMQ para confirmar que inició correctamente
kubectl logs -l app=rabbitmq --tail=20

# Buscar la línea: "Server startup complete"

# Hacer port-forward para acceder a la interfaz de administración
kubectl port-forward svc/rabbitmq 15672:15672 &

# Abrir en navegador: http://localhost:15672
# Usuario: admin
# Password: admin123

# Cerrar port-forward
pkill -f "port-forward svc/rabbitmq"
```

**Salida esperada:**
- Deployment: `1/1 replicas` listas
- Pod logs: "Server startup complete"
- Services: puerto 5672 (AMQP) y 15672 (Management UI)
- URL dentro de la red del cluster: `rabbitmq:5672`

---

### PASO 4: Desplegar Servicios gRPC (Receptor y Publicadores)

#### 4a. Desplegar Receptor (Enrutador)

```bash
kubectl apply -f yaml/grcp/go-receptor-deployment.yaml
```

**Verificación:**
```bash
# Ver el deployment
kubectl get deployment receptor

# Ver el pod
kubectl get pods -l app=receptor

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=receptor --timeout=120s

# Ver logs para confirmar conexión con publicadores
kubectl logs -l app=receptor --tail=30
```

**Salida esperada en logs:**
```
✅ Conectado al Servidor (RabbitMQ) en :publisher-rabbit-service:50052
✅ Conectado al Servidor (Kafka) en :publisher-kafka-service:50053
✅ SERVIDOR 1 (Receptor) listo
```

---

#### 4b. Desplegar Publicador RabbitMQ

```bash
kubectl apply -f yaml/grcp/go-rabbitmq-writer-deployment.yaml
```

**Verificación:**
```bash
# Ver el deployment
kubectl get deployment publisher-rabbit

# Ver el pod
kubectl get pods -l app=publisher-rabbit

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=publisher-rabbit --timeout=120s

# Ver logs para confirmar conexión con RabbitMQ
kubectl logs -l app=publisher-rabbit --tail=30
```

**Prueba de conectividad:**
```bash
# Verificar que el publicador se conectó a RabbitMQ
kubectl logs -l app=publisher-rabbit | grep "RabbitMQ configurado"

# Debe mostrar:
# ✅ RabbitMQ configurado correctamente
```

---

#### 4c. Desplegar Publicador Kafka

```bash
kubectl apply -f yaml/grcp/go-kafka-writer-deployment.yaml
```

**Verificación:**
```bash
# Ver el deployment
kubectl get deployment publisher-kafka

# Ver el pod
kubectl get pods -l app=publisher-kafka

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=publisher-kafka --timeout=120s

# Ver logs para confirmar conexión con Kafka
kubectl logs -l app=publisher-kafka --tail=30
```

**Prueba de conectividad:**
```bash
# Verificar que el publicador se conectó a Kafka
kubectl logs -l app=publisher-kafka | grep "Conectado al broker"

# Debe mostrar:
# ✅ Conectado al broker de Kafka en my-cluster-kafka-bootstrap:9092
```

---

### PASO 5: Desplegar API REST en Rust

```bash
kubectl apply -f yaml/rust-api/rust-api-deployment.yaml
kubectl apply -f yaml/rust-api/rust-api-service.yaml
```

**Verificación:**
```bash
# Ver el deployment
kubectl get deployment rust-api

# Ver el pod
kubectl get pods -l app=rust-api

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=rust-api --timeout=120s

# Ver el servicio e ingress
kubectl get svc rust-api-service
kubectl get ingress rust-api-ingress
```

**Prueba de conectividad:**
```bash
# Probar endpoint desde dentro del cluster
kubectl run test-api --rm -it --image=curlimages/curl -- curl http://rust-api-service:8080/

# Debe responder: "hola"

# Ver logs para confirmar conexión con receptor
kubectl logs -l app=rust-api --tail=20
```

**Salida esperada en logs:**
```
Variable GRPC_SERVER_URL cargada: http://receptor-service:50051
Rust REST API corriendo en http://0.0.0.0:8080
```

---

### PASO 6: Desplegar Consumidores

#### 6a. Desplegar Consumidor RabbitMQ

```bash
kubectl apply -f yaml/consumers/rabbit-consumer-deployment.yaml
```

**Verificación:**
```bash
# Ver el deployment
kubectl get deployment consumer-rabbit

# Ver el pod
kubectl get pods -l app=consumer-rabbit

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=consumer-rabbit --timeout=120s

# Ver logs para confirmar conexión
kubectl logs -l app=consumer-rabbit --tail=30
```

**Prueba de conectividad:**
```bash
# Verificar conexión con RabbitMQ y Valkey
kubectl logs -l app=consumer-rabbit | grep -E "(RabbitMQ|Valkey)"

# Debe mostrar:
# 🔄 Consumidor de RabbitMQ iniciado
# ✅ Conectado a Valkey: PONG
```

---

#### 6b. Desplegar Consumidor Kafka

```bash
kubectl apply -f yaml/consumers/kafka-consumer-deployment.yaml
```

**Verificación:**
```bash
# Ver el deployment
kubectl get deployment kafka-consumer

# Ver el pod
kubectl get pods -l app=kafka-consumer

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=kafka-consumer --timeout=120s

# Ver logs para confirmar conexión
kubectl logs -l app=kafka-consumer --tail=30
```

**Prueba de conectividad:**
```bash
# Verificar conexión con Kafka y Valkey
kubectl logs -l app=kafka-consumer | grep -E "(Kafka|Valkey)"

# Debe mostrar:
# 🚀 Iniciando consumidor de Kafka...
# ✅ Consumidor conectado a Kafka y listo para recibir mensajes...
```

---

### 📊 PASO 7: Desplegar Grafana (Monitoreo)

```bash
kubectl apply -f yaml/grafana-deplyment.yaml
```

**Verificación:**
```bash
# Ver el deployment
kubectl get deployment grafana

# Ver el pod
kubectl get pods -l app=grafana

# Esperar a que esté listo
kubectl wait --for=condition=Ready pod -l app=grafana --timeout=120s

# Ver el servicio
kubectl get svc grafana-service

kubectl get pods -l app=grafana

kubectl port-forward grafana-5fdbc49554-h2wlx 8080:3000
```

**Acceso a Grafana:**
```bash
# Obtener IP de un nodo
kubectl get nodes -o wide

# Acceder en navegador: http://<NODE_IP>:30300
# Usuario: admin
# Password: admin
```

---

## PRUEBA COMPLETA DEL FLUJO DE DATOS

### 1. Verificar que todos los pods estén corriendo

```bash
kubectl get pods
```

**Salida esperada:** Todos los pods en estado `Running` con `READY 1/1`

---

### 2. Obtener IP externa del Ingress

```bash
kubectl get ingress rust-api-ingress

# Esperar a que aparezca la ADDRESS (puede tardar 1-2 minutos)
```

---

### 3. Enviar petición de prueba

```bash
# Usar la IP del ingress
INGRESS_IP=$(kubectl get ingress rust-api-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Enviar datos de clima
curl -X POST http://${INGRESS_IP}/clima \
  -H "Content-Type: application/json" \
  -d '{
    "municipality": 1,
    "temperature": 25,
    "humidity": 70,
    "weather": 1
  }'
```

**Respuesta esperada:** 
```json
{"status":"Tweet enviado a RabbitMQ/Kafka correctamente ✅","message_id":"..."}
```

---

### 4. Verificar flujo en los logs

```bash
# Ver logs de cada componente para seguir el flujo

# 1. Rust API recibe la petición
kubectl logs -l app=rust-api --tail=5

# 2. Receptor enruta a un publicador
kubectl logs -l app=receptor --tail=10

# 3. Publicador envía a broker (RabbitMQ o Kafka)
kubectl logs -l app=publisher-rabbit --tail=10
kubectl logs -l app=publisher-kafka --tail=10

# 4. Consumidor recibe y procesa
kubectl logs -l app=consumer-rabbit --tail=10
kubectl logs -l app=kafka-consumer --tail=10
```

---

### 5. Verificar datos en Valkey

```bash
# Conectar a Valkey
kubectl exec -it deployment/valkey -- valkey-cli

# Dentro de valkey-cli:
KEYS *
# Debe mostrar: municipality:mixco (o el municipio enviado)

HGETALL municipality:mixco
# Debe mostrar: name, temperature, humidity, weather, last_update

# Salir
exit
```

---

## 🔍 VERIFICACIÓN DE CONECTIVIDAD ENTRE COMPONENTES

### Verificar conectividad: Rust API → Receptor

```bash
kubectl run test-grpc --rm -it --image=alpine -- sh

# Dentro del pod:
apk add --no-cache curl
curl -v receptor-service:50051
# Debe intentar conectar (error esperado por ser gRPC)

exit
```

---

### Verificar conectividad: Publicadores → RabbitMQ

```bash
# Ver que el publicador RabbitMQ puede comunicarse
kubectl exec -it deployment/publisher-rabbit -- sh

# Dentro del pod (si tiene herramientas):
ping rabbitmq
# O simplemente revisar los logs que ya verificamos
exit
```

---

### Verificar conectividad: Publicadores → Kafka

```bash
# Crear tema de prueba desde el publicador Kafka
kubectl run kafka-test --rm -it --image=quay.io/strimzi/kafka:latest-kafka-3.8.0 -- \
  bin/kafka-topics.sh --bootstrap-server my-cluster-kafka-bootstrap:9092 \
  --describe --topic weather_tweets

# Debe mostrar el topic con sus particiones
```

---

### Verificar conectividad: Consumidores → Valkey

```bash
# Desde el consumidor RabbitMQ
kubectl run test-redis --rm -it --image=redis:alpine -- \
  redis-cli -h valkey-service -p 6379 PING

# Debe responder: PONG
```

---

## 📈 MONITOREO CONTINUO

### Ver logs en tiempo real de todos los componentes

```bash
# Opción 1: Ver todos los logs
kubectl logs -f -l app=rust-api &
kubectl logs -f -l app=receptor &
kubectl logs -f -l app=consumer-rabbit &

# Opción 2: Usar stern (si está instalado)
# brew install stern
stern "rust-api|receptor|consumer"
```

---

### Ver recursos del cluster

```bash
# Uso de CPU y memoria por nodo
kubectl top nodes

# Uso de CPU y memoria por pod
kubectl top pods

# Ver todos los servicios
kubectl get svc
```

---

## ⚠️ SOLUCIÓN DE PROBLEMAS

### Si un pod no inicia (Pending/CrashLoopBackOff)

```bash
# Ver detalles del pod
kubectl describe pod <nombre-del-pod>

# Ver logs del pod
kubectl logs <nombre-del-pod>

# Ver logs del contenedor anterior (si crasheó)
kubectl logs <nombre-del-pod> --previous
```

---

### Si hay problemas de conectividad

```bash
# Verificar que los servicios tengan endpoints
kubectl get endpoints

# Probar DNS interno
kubectl run test-dns --rm -it --image=busybox -- nslookup rust-api-service

# Ver reglas de red
kubectl get networkpolicies
```

---

### Reiniciar un componente específico

```bash
# Reiniciar deployment
kubectl rollout restart deployment <nombre-deployment>

# Eliminar y recrear un pod
kubectl delete pod <nombre-pod>
# Kubernetes lo recreará automáticamente
```

---

## 🧹 LIMPIEZA COMPLETA

```bash
# Eliminar en orden inverso
kubectl delete -f yaml/grafana-deplyment.yaml
kubectl delete -f yaml/consumers/
kubectl delete -f yaml/rust-api/
kubectl delete -f yaml/grcp/
kubectl delete -f yaml/brokers/
kubectl delete -f yaml/valkey-deployment.yaml

# Eliminar el operador Strimzi
kubectl delete -f 'https://strimzi.io/install/latest?namespace=default'

# Verificar que todo se eliminó
kubectl get all
```

---

## ✅ CHECKLIST DE DESPLIEGUE COMPLETO

- [ ] Operador Strimzi instalado y corriendo
- [ ] Valkey desplegado y respondiendo a PING
- [ ] Kafka cluster activo con bootstrap service
- [ ] RabbitMQ activo con management UI accesible
- [ ] Receptor conectado a ambos publicadores
- [ ] Publicador RabbitMQ conectado al broker
- [ ] Publicador Kafka conectado al broker
- [ ] Rust API respondiendo en endpoint `/`
- [ ] Consumidor RabbitMQ conectado a RabbitMQ y Valkey
- [ ] Consumidor Kafka conectado a Kafka y Valkey
- [ ] Grafana accesible en puerto 30300
- [ ] Ingress con IP externa asignada
- [ ] Prueba end-to-end exitosa (POST a `/clima`)
- [ ] Datos verificados en Valkey

---

**🎉 ¡Despliegue Completado!**

Tu sistema de microservicios está listo para recibir y procesar datos de clima en tiempo real.
