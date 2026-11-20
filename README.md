# Operating Systems 1 — Linux kernel, containers and Kubernetes

Lab work for *Sistemas Operativos 1* (Universidad de San Carlos de Guatemala, 2nd semester 2025).
Individual repository. It goes from the Linux kernel up to a microservices system on Google Kubernetes Engine.

| Folder | What it is | Stack |
|---|---|---|
| [`proyecto-2/`](proyecto-2/) | Container resource monitor: two kernel modules, a Go daemon and a Grafana dashboard | C (kernel modules), Go, SQLite, Docker, Bash, Grafana |
| [`proyecto-3/`](proyecto-3/) | "WeatherTweet": event pipeline on GKE with gRPC, Kafka and RabbitMQ | Rust (Actix), Go, gRPC, Kafka (Strimzi), RabbitMQ, Valkey, Grafana, Locust, Kubernetes |
| [`proyecto1/`](proyecto1/) | Three Go APIs packaged as images, pushed to a private Zot registry and run with containerd across KVM virtual machines | Go, Docker, containerd, Zot, libvirt/virsh |
| [`practica-unica/`](practica-unica/) | Zot registry running on Kubernetes with a persistent volume | Kubernetes |
| [`tarea-1/`](tarea-1/) | Linux command-line basics and a container-simulation script | Bash |

## Project 2 — container monitor from the kernel

```text
create.sh (cron, every minute) ──► starts CPU / RAM / low-load Python containers
                                              │
 /proc/sysinfo_so1_202300350  ◄── sysinfo_so1.ko  (RAM, process list, CPU %, state)
 /proc/continfo_so1_202300350 ◄── continfo_so1.ko (per-container processes)
                                              │
 Go daemon (every 20 s) ── reads /proc ──► SQLite ──► Grafana dashboard
          └── keeps the 5 newest stress containers and removes the rest
```

- **Kernel modules (C):** walk `task_struct` to report memory (VSZ/RSS), CPU and memory percentages, state and command line of each process, and expose them as JSON in `/proc`.
- **Go daemon:** on start it loads the modules and installs the cron job. Every 20 s it stores the readings in SQLite and removes extra containers. On SIGINT/SIGTERM it removes the cron job, unloads the modules and stops the container it started.
- **Stress containers:** three Python images (CPU, RAM, low load) launched at random by `create.sh`.

Technical manual: [`proyecto-2/docs/README.md`](proyecto-2/docs/README.md).

## Project 3 — WeatherTweet on Kubernetes

```text
Locust ──HTTP──► Rust API (Actix) ──gRPC──► Go receptor ──gRPC──┬──► Kafka writer ──► Kafka ──► Kafka consumer ──┐
                                                                └──► Rabbit writer ──► RabbitMQ ──► Rabbit consumer ─┤
                                                                                                                     ▼
                                                                                                    Valkey ──► Grafana
```

- **Rust API:** receives `POST /clima` (municipality, temperature, humidity, weather) and forwards it over gRPC (`tonic`).
- **Go services:** one receptor and two publishers share the `weathertweet.proto` contract. The consumers store the messages in Valkey, which Grafana reads.
- **Kubernetes:** manifests for every service, Kafka through the Strimzi operator, and an Ingress for the API. The resource requests were tuned to fit a 3-node `e2-medium` cluster: 38% less CPU and 42% less RAM requested ([`RESOURCE_OPTIMIZATION.md`](proyecto-3/RESOURCE_OPTIMIZATION.md)).
- **Load testing:** Locust scenarios in [`proyecto-3/src/locust`](proyecto-3/src/locust).

Deployment: [`DEPLOYMENT_GUIDE.md`](proyecto-3/DEPLOYMENT_GUIDE.md) (English, short) and [`DESPLIEGUE_PASO_A_PASO.md`](proyecto-3/DESPLIEGUE_PASO_A_PASO.md) (Spanish, with checks at each step).

## Notes

- The Grafana and registry passwords in these files (`admin`/`admin123`) are local lab defaults, not real credentials.
- Some scripts use absolute paths from my machine (`/home/abdielo/...`). Change them before running.
