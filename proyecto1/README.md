
# 🚀 Gestión de Máquinas Virtuales y Contenedores

Este documento proporciona una guía completa para la gestión y operación de máquinas virtuales con contenedores distribuidos usando virsh, Docker y containerd.

## 📋 Tabla de Contenidos

-   [Obtener Direcciones IP](#-obtener-direcciones-ip)
-   [Conexión SSH a VMs](#-conexi%C3%B3n-ssh-a-vms)
-   [Gestión de Servicios](#%EF%B8%8F-gesti%C3%B3n-de-servicios)
-   [Despliegue de APIs](#-despliegue-de-apis)
-   [Pruebas y Validación](#-pruebas-y-validaci%C3%B3n)

## 🌐 Obtener Direcciones IP

Para verificar las direcciones IP asignadas a las máquinas virtuales:

```bash
# VM1
virsh domifaddr vm1

# VM2
virsh domifaddr vm2

# Ubuntu 22.04
virsh domifaddr ubuntu22.04

```

## 🔐 Conexión SSH a VMs

Conectarse a las diferentes máquinas virtuales usando SSH:

```bash
# DockerVM (Servidor de registro)
ssh abdielo@192.168.122.123

# VM1 (APIs 1 y 2)
ssh abdielo@192.168.122.173

# VM2 (API 3)
ssh abdielo@192.168.122.17

```

## ⚙️ Gestión de Servicios

### Iniciar Servicio Zot (Registro de Contenedores)

En la DockerVM, iniciar el servicio de registro Zot:

```bash
sudo docker start 983f

```

### Verificar Catálogo de Imágenes

Para ver las imágenes disponibles en el registro:

```bash
curl localhost:5000/v2/_catalog

```

## 🚀 Despliegue de APIs

### En VM1 (192.168.122.173)

Desplegar API1 y API2:

```bash
# Iniciar API1
sudo ctr run -d --net-host 192.168.122.123:5000/api1:vm1 vm1-api1

# Iniciar API2
sudo ctr run -d --net-host 192.168.122.123:5000/api2:vm1 vm1-api2

```

### En VM2 (192.168.122.17)

Desplegar API3:

```bash
# Iniciar API3
sudo ctr run -d --net-host 192.168.122.123:5000/api3:vm2 vm2-api3

```

## 🧪 Pruebas y Validación

### URLs de Prueba para API1 (Puerto 3001)

```bash
# Endpoint principal
http://192.168.122.173:3001/

# Llamadas inter-API desde API1
http://192.168.122.173:3001/api1/202300350/llamar-api2
http://192.168.122.173:3001/api1/202300350/llamar-api3

```

### URLs de Prueba para API2 (Puerto 3002)

```bash
# Endpoint principal
http://192.168.122.173:3002/

# Llamadas inter-API desde API2
http://192.168.122.173:3002/api2/202300350/llamar-api1
http://192.168.122.173:3002/api2/202300350/llamar-api3

```

### URLs de Prueba para API3 (Puerto 3003)

```bash
# Endpoint principal
http://192.168.122.17:3003/

# Llamadas inter-API desde API3
http://192.168.122.17:3003/api3/202300350/llamar-api1
http://192.168.122.17:3003/api3/202300350/llamar-api2

```

## 📊 Arquitectura del Sistema

```
DockerVM (192.168.122.123)
├── Zot Registry (Puerto 5000)
└── Imágenes de contenedores

VM1 (192.168.122.173)
├── API1 (Puerto 3001)
└── API2 (Puerto 3002)

VM2 (192.168.122.17)
└── API3 (Puerto 3003)

```

## 💡 Notas Importantes

-   **Usuario SSH**: `abdielo` para todas las conexiones
-   **ID de Estudiante**: `202300350` utilizado en las rutas de las APIs
-   **Registro de Contenedores**: Centralizado en DockerVM usando Zot
-   **Networking**: Todas las APIs usan `--net-host` para comunicación directa

## 🔧 Troubleshooting

Si encuentras problemas:

1.  Verifica que todas las VMs estén ejecutándose
2.  Confirma la conectividad SSH antes del despliegue
3.  Asegúrate de que el servicio Zot esté activo
4.  Verifica que las imágenes estén disponibles en el catálogo