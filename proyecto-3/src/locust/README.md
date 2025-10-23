# Guía de Pruebas de Carga con Locust

## Instalación de Locust

```bash
# Opción 1: Con pip
pip install locust

# Opción 2: Con pip3
pip3 install locust

# Verificar instalación
locust --version
```

---

## Ejecutar Pruebas de Carga

### 1. Modo Interfaz Web (Recomendado)

```bash
cd src/locust

# Ejecutar Locust con interfaz web
locust -f main.py

# O especificar un puerto diferente
locust -f main.py --web-port 8089
```

**Acceso:**
1. Abrir navegador en: `http://localhost:8089`
2. Configurar:
   - **Number of users:** 10 (usuarios concurrentes)
   - **Spawn rate:** 2 (usuarios nuevos por segundo)
   - **Host:** `http://34.36.85.14` (ya está configurado por defecto)
3. Click en **Start swarming**

---

### 2. Modo Headless (Sin interfaz, para CI/CD)

```bash
cd src/locust

# Prueba rápida: 10 usuarios, 2 por segundo, 30 segundos
locust -f main.py --headless -u 10 -r 2 -t 30s --host http://34.36.85.14

# Prueba de carga media: 50 usuarios, 5 por segundo, 2 minutos
locust -f main.py --headless -u 50 -r 5 -t 2m --host http://34.36.85.14

# Prueba de estrés: 100 usuarios, 10 por segundo, 5 minutos
locust -f main.py --headless -u 100 -r 10 -t 5m --host http://34.36.85.14

# Prueba extrema: 200 usuarios, 20 por segundo, 10 minutos
locust -f main.py --headless -u 200 -r 20 -t 10m --host http://34.36.85.14
```

**Parámetros:**
- `-u, --users`: Número total de usuarios concurrentes
- `-r, --spawn-rate`: Usuarios nuevos por segundo
- `-t, --run-time`: Duración de la prueba (30s, 2m, 1h)
- `--host`: URL del servidor a probar

---

### 3. Generar Reporte HTML

```bash
# Ejecutar con reporte
locust -f main.py --headless -u 50 -r 5 -t 2m \
  --host http://34.36.85.14 \
  --html report.html \
  --csv results

# Esto generará:
# - report.html: Reporte visual con gráficas
# - results_stats.csv: Estadísticas de requests
# - results_failures.csv: Logs de errores
# - results_stats_history.csv: Historial de estadísticas
```

---

## Escenarios de Prueba

### Prueba de Smoke (Verificación Básica)
```bash
# 5 usuarios, 1 por segundo, 30 segundos
locust -f main.py --headless -u 5 -r 1 -t 30s --host http://34.36.85.14
```
**Objetivo:** Verificar que el sistema funciona bajo carga mínima

---

### Prueba de Carga Normal
```bash
# 25 usuarios, 3 por segundo, 3 minutos
locust -f main.py --headless -u 25 -r 3 -t 3m --host http://34.36.85.14
```
**Objetivo:** Simular uso normal del sistema

---

### Prueba de Estrés
```bash
# 100 usuarios, 10 por segundo, 5 minutos
locust -f main.py --headless -u 100 -r 10 -t 5m --host http://34.36.85.14
```
**Objetivo:** Encontrar el límite del sistema

---

### Prueba de Picos (Spike Test)
```bash
# Rápidamente llegar a 150 usuarios
locust -f main.py --headless -u 150 -r 50 -t 2m --host http://34.36.85.14
```
**Objetivo:** Ver cómo responde el sistema a picos repentinos

---

### Prueba de Resistencia (Soak Test)
```bash
# 30 usuarios constantes por 30 minutos
locust -f main.py --headless -u 30 -r 3 -t 30m --host http://34.36.85.14
```
**Objetivo:** Detectar memory leaks y degradación en el tiempo

---

## Monitoreo Durante las Pruebas

### 1. Ver métricas de Kubernetes en tiempo real

```bash
# Terminal 1: Ver uso de recursos
watch kubectl top pods

# Terminal 2: Ver logs del Rust API
kubectl logs -f deployment/rust-api --tail=50

# Terminal 3: Ver logs del receptor
kubectl logs -f deployment/receptor --tail=50

# Terminal 4: Ver logs de consumidores
kubectl logs -f deployment/consumer-rabbit --tail=50
```

---

### 2. Verificar datos en Valkey durante la prueba

```bash
# Conectar a Valkey
VALKEY_POD=$(kubectl get pods -l app=valkey -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $VALKEY_POD -- valkey-cli

# Dentro de valkey-cli, ejecutar cada 5 segundos:
KEYS *
DBSIZE
INFO stats
```

---

### 3. Verificar colas de RabbitMQ

```bash
# Port-forward a RabbitMQ
kubectl port-forward svc/rabbitmq 15672:15672 &

# Acceder a: http://localhost:15672
# Usuario: admin, Password: admin123
# Ver Queues > weather_tweets_queue
```

---

## Interpretación de Resultados

### Métricas Clave en Locust:

1. **Request/s (RPS)**: Requests por segundo
   - 🟢 Bueno: >50 RPS sin errores
   - 🟡 Aceptable: 20-50 RPS
   - 🔴 Malo: <20 RPS o con errores

2. **Response Time**:
   - 🟢 Excelente: <100ms (p95)
   - 🟡 Aceptable: 100-500ms (p95)
   - 🔴 Lento: >500ms (p95)

3. **Failure Rate**:
   - 🟢 Perfecto: 0%
   - 🟡 Aceptable: <1%
   - 🔴 Problemático: >5%

---

## Ejemplo de Salida en Modo Headless

```
[2025-10-22 10:30:00,000] INFO/locust.main: Starting Locust 2.x
[2025-10-22 10:30:00,000] INFO/locust.runners: Ramping to 50 users at a rate of 5.00 per second
Type     Name                              # reqs      # fails |    Avg     Min     Max    Med |   req/s  failures/s
--------|--------------------------------|-------|-------------|-------|-------|-------|-------|--------|-----------
POST     /clima                            2450     0(0.00%)  |     45      12     234     38 |   40.83        0.00
--------|--------------------------------|-------|-------------|-------|-------|-------|-------|--------|-----------
         Aggregated                        2450     0(0.00%)  |     45      12     234     38 |   40.83        0.00

Response time percentiles (approximated)
Type     Name                              50%    66%    75%    80%    90%    95%    98%    99%  99.9% 99.99%   100% # reqs
--------|--------------------------------|--------|------|------|------|------|------|------|------|------|------|------|------
POST     /clima                              38     42     48     52     65     78     95    110    200    230    234   2450
--------|--------------------------------|--------|------|------|------|------|------|------|------|------|------|------|------
         Aggregated                          38     42     48     52     65     78     95    110    200    230    234   2450
```

---

## Consejos para Pruebas Efectivas

### 1. Preparación
```bash
# Antes de empezar, asegúrate que todo esté corriendo:
kubectl get pods
kubectl get svc
curl http://34.36.85.14/  # Debe responder "hola"
```

### 2. Durante la Prueba
- Monitorea recursos de Kubernetes: `kubectl top nodes`
- Revisa logs en tiempo real para detectar errores
- Observa las métricas de Locust en el navegador

### 3. Después de la Prueba
```bash
# Verificar que los datos llegaron a Valkey
kubectl exec -it $VALKEY_POD -- valkey-cli DBSIZE

# Revisar logs de errores
kubectl logs deployment/rust-api | grep -i error
kubectl logs deployment/receptor | grep -i error
```

---

## Automatización con Docker (Opcional)

Si quieres ejecutar Locust en un contenedor:

```dockerfile
# Crear archivo: src/locust/Dockerfile
FROM locustio/locust

COPY main.py /mnt/locust/
```

```bash
# Build y run
cd src/locust
docker build -t weather-locust .

# Ejecutar con interfaz web
docker run -p 8089:8089 weather-locust -f /mnt/locust/main.py --host http://34.36.85.14

# Ejecutar headless
docker run weather-locust -f /mnt/locust/main.py --headless -u 50 -r 5 -t 2m --host http://34.36.85.14
```

---

## Solución de Problemas

### Error: "Connection refused"
```bash
# Verificar que la IP/URL es correcta
curl http://34.36.85.14/

# Si no responde, verificar ingress
kubectl get ingress
```

### Error: "Too many open files"
```bash
# Aumentar límite de archivos (Linux/Mac)
ulimit -n 10000
```

### Locust muy lento
```bash
# Usar más workers (distribuido)
locust -f main.py --master &
locust -f main.py --worker &
locust -f main.py --worker &
```

---

## 🎯 Prueba Recomendada Inicial

```bash
cd src/locust

# Prueba progresiva de 5 minutos
locust -f main.py --headless \
  -u 50 -r 5 -t 5m \
  --host http://34.36.85.14 \
  --html report_$(date +%Y%m%d_%H%M%S).html

# Revisar el reporte HTML generado
```

Esta prueba te dará una buena idea del rendimiento del sistema sin saturarlo. 🚀
