# Manuel Técnico

**Organizacion de archivos y directorios**
-
- bash: en esta carpeta se encuentran los scripts y archivos relacionados con el entorno de bash. como los siguientes:
1. mod_manager.sh: Script para gestionar la carga y descarga de módulos del kernel.
2. script.sh: Script de ejemplo para demostrar el uso de bash.

- dashboard: en esta carpeta se encuentran los archivos relacionados con el dashboard de Grafana y el daemon en Go.
1. daemon: El archivo compilado del daemon en Go.
2. main.go: El código fuente del daemon en Go.
3. grafana/: Carpeta que contiene los archivos de configuración y paneles de Grafana.

- docker: en esta carpeta se encuentran los archivos relacionados con Docker.
1. cpu/: Dockerfile y scripts para el contenedor de consumo de CPU.
2. low/: Dockerfile y scripts para el contenedor de baja consumo.
3. ram/: Dockerfile y scripts para el contenedor de consumo de RAM.
4. clean.sh: Script para limpiar los contenedores generados.
5. create.sh: Script para crear los contenedores 10 contenedores de CPU, RAM y bajo consumo.

- docs: en esta carpeta se encuentran los documentos relacionados con el proyecto.
1. README.md: Este archivo.

- go-daemon: en esta carpeta se encuentran los archivos relacionados con el daemon en Go.
1. main.go: El código fuente del daemon en Go.
2. go.mod: Archivo de módulos de Go.
3. go.sum: Archivo de sumas de verificación de Go.

- modulo-kernel: en esta carpeta se encuentran los archivos relacionados con el módulo del kernel.
1. Makefile: Archivo de make para compilar el módulo del kernel.
2. sysinfo_so1.c: Código fuente del módulo del kernel que recopila información del sistema.
3. continfo_so1.c: Código fuente del módulo del kernel que recopila información de contenedores Docker.

**Estructura del módulo**
Para compilar el módulo del kernel, se utiliza un Makefile que define las reglas para compilar el código fuente del módulo. El módulo principal es `sysinfo_so1.c`, que recopila información del sistema y la expone a través del sistema de archivos `/proc`. Además, hay un segundo módulo `continfo_so1.c` que recopila información específica de los contenedores Docker.

Para compilar el módulo, se utiliza el comando `make` en la terminal dentro del directorio `modulo-kernel`. Esto generará un archivo `.ko` que puede ser cargado en el kernel utilizando el comando `insmod`.

para compilar el módulo del kernel:
```bash
make
```

Para limpiar los archivos generados:
```bash
make clean
```

**Funciones principales del módulo de kernel**


## Funciones Principales

### `get_process_cmdline(struct task_struct *task)`

Extrae la línea de comandos completa de un proceso específico desde su memoria virtual.

-   Accede a la memoria virtual del proceso
-   Reemplaza caracteres nulos por espacios
-   Retorna un puntero a la cadena de comandos

### `get_task_state(struct task_struct *task)`

Convierte el estado del proceso a una representación legible.

-   Estados soportados: RUNNING, SLEEPING, WAITING, STOPPED, TRACING_STOP, ZOMBIE, DEAD, UNKNOWN

### `sysinfo_show(struct seq_file *m, void *v)`

Función principal que genera la salida JSON con información del sistema:

-   **Información de memoria:** RAM total y libre
-   **Conteo de procesos activos**
-   **Lista detallada de procesos** con:
    -   PID, nombre y línea de comandos
    -   Memoria virtual (VSZ) y física (RSS)
    -   Porcentaje de uso de memoria y CPU
    -   Estado del proceso

### `sysinfo_open(struct inode *inode, struct file *file)`

Maneja la apertura del archivo `/proc` utilizando `single_open()`.

## Funciones de Control del Módulo

### `sysinfo_init()`

-   Crea la entrada `/proc/sysinfo_so1_202300350`
-   Registra las operaciones del archivo proc

### `sysinfo_exit()`

-   Elimina la entrada del directorio `/proc`
-   Limpia los recursos del módulo

# Instrucciones para compilar y cargar el módulo del kernel
----
Quitar el módulo actual:

```bash
sudo rmmod sysinfo_so1
```


Revisar que ya no esté:
```bash
lsmod | grep sysinfo_so1
```


Limpiar y recompilar:
```bash
make clean
make
```


Cargar el módulo nuevo:
```bash
sudo insmod sysinfo_so1.ko
```


Verificar en /proc:
```bash
cat /proc/sysinfo_so1_202300350
cat /proc/continfo_so1_202300350
```

# Instrucciones para compilar y ejecutar el daemon en Go

compilar el daemon:
```bash
go build -o daemon main.go
```

Crear un archivo de servicio
```bash
sudo nano /etc/systemd/system/grafana-db-daemon.service
```

contenido:
```ini  
[Unit]
Description=Daemon en Go grafana
After=network.target

[Service]
ExecStart=/home/abdielo/dev/sopes/proyecto1/proyecto-2/dashboard/daemon
WorkingDirectory=/home/abdielo/dev/sopes/proyecto1/proyecto-2/dashboard
Restart=always

[Install]
WantedBy=multi-user.target
```
Guardar y salir.
Recargar los archivos de servicio:
```bash
sudo systemctl daemon-reload
sudo systemctl enable grafana-db-daemon
sudo systemctl start grafana-db-daemon
```

Verificar el estado del servicio:
```bash
sudo systemctl status grafana-db-daemon
```

quitar el servicio:
```bash
sudo systemctl stop grafana-db-daemon
sudo systemctl disable grafana-db-daemon
sudo rm /etc/systemd/system/grafana-db-daemon.service
sudo systemctl daemon-reload
```

script completo para volver a compilar el módulo y reiniciar el daemon:
```bash
sudo systemctl stop grafana-db-daemon
sudo systemctl disable grafana-db-daemon
sudo rm daemon
go build -o daemon main.go
sudo chcon -t bin_t /home/abdielo/dev/sopes/proyecto1/proyecto-2/dashboard/daemon
sudo chmod +x /home/abdielo/dev/sopes/proyecto1/proyecto-2/dashboard/daemon
sudo systemctl daemon-reload
sudo systemctl enable grafana-db-daemon
sudo systemctl start grafana-db-daemon
``` 
    
status y logs:
```bash
sudo systemctl status grafana-db-daemon
journalctl -u grafana-db-daemon -f
```

Para que el daemon puede ejecutar el script es importante desactivar el SELinux o configurar los contextos de seguridad adecuados.
```bash
sudo setenforce 0
```
y cuando se haya terminado (para volver a activar)
```bash
sudo setenforce 1
```

## Estructura del Daemon GO
----
## Descripción

Daemon desarrollado en Go que monitorea el sistema operativo y gestiona contenedores Docker automáticamente. Recopila información de procesos usando módulos de kernel personalizados y almacena los datos en una base de datos SQLite.

## Estructuras de Datos

### `Container`

Representa un contenedor simulado:

-   `Name`: Nombre del contenedor
-   `CPU`: Uso de CPU
-   `Memory`: Uso de memoria
-   `Status`: Estado del contenedor

### `Proccess`

Información detallada de un proceso:

-   `PID`: ID del proceso
-   `Name`: Nombre del proceso
-   `Cmdline`: Línea de comandos
-   `vsz`: Memoria virtual
-   `rss`: Memoria física
-   `Memory_Usage`: Porcentaje de uso de memoria
-   `CPU_Usage`: Porcentaje de uso de CPU
-   `State`: Estado del proceso

### `Lecture`

Estructura principal que contiene:

-   `Totalram`: RAM total del sistema
-   `Freeram`: RAM libre
-   `Procs`: Número de procesos
-   `Processes`: Lista de procesos

## Funciones Principales

### `main()`

Función principal del daemon que:

-   Configura y ejecuta un cronjob para crear contenedores
-   Inicia contenedor de Grafana
-   Carga módulos de kernel personalizados
-   Inicializa base de datos SQLite con tres tablas
-   Ejecuta bucle principal de monitoreo cada 20 segundos
-   Maneja señales del sistema para cierre controlado

### `readSysinfo()`

Lee información del sistema desde `/proc/sysinfo_so1_202300350`:

-   Ejecuta comando `cat` en el archivo proc
-   Deserializa el JSON retornado
-   Retorna estructura `Lecture` con información del sistema

### `readContinfo()`

Lee información de contenedores desde `/proc/continfo_so1_202300350`:

-   Similar a `readSysinfo()` pero enfocado en contenedores
-   Retorna estructura `Lecture` con datos de procesos de contenedores

### `getRunningPythonContainerIDs()`

Obtiene IDs de contenedores Python en ejecución:

-   Ejecuta comando Docker con filtros para contenedores corriendo
-   Filtra por imagen "python" usando grep
-   Retorna slice de IDs de contenedores

## Funcionalidades del Sistema

### Gestión de Base de Datos

Crea tres tablas en SQLite:

1.  **`containers`**: Información básica de contenedores
2.  **`continfo`**: Información detallada de procesos
3.  **`sysinfo`**: Información general del sistema

### Gestión Automática de Contenedores

-   **Creación**: Cronjob ejecuta script cada minuto para crear contenedores
-   **Eliminación inteligente**:
    -   Si hay más de 5 contenedores Python corriendo
    -   Elimina aleatoriamente la mitad de los contenedores más antiguos
    -   Mantiene siempre los últimos 5 contenedores

### Monitoreo Continuo

El daemon ejecuta cada 20 segundos:

1.  Lee información de procesos desde módulo de kernel
2.  Inserta datos en tabla `continfo`
3.  Lee información general del sistema
4.  Inserta datos en tabla `sysinfo`
5.  Evalúa y gestiona contenedores Docker

### Inicialización del Sistema

Al arrancar:

-   Configura permisos y crea cronjob
-   Inicia contenedor de Grafana (ID: 7518b64c3ca3)
-   Carga módulos de kernel usando script `mod_manager.sh`

### Limpieza al Cierre

Al recibir señales SIGINT/SIGTERM:

-   Elimina todos los cronjobs
-   Descarga módulos de kernel
-   Detiene contenedor de Grafana
-   Cierre controlado del daemon

## Dependencias

-   `github.com/mattn/go-sqlite3`: Driver SQLite para Go
-   Módulos de kernel personalizados en `/proc/`
-   Docker para gestión de contenedores
-   Scripts bash auxiliares

## Archivos del Sistema

-   `/proc/sysinfo_so1_202300350`: Información del sistema desde módulo kernel
-   `/proc/continfo_so1_202300350`: Información de contenedores desde módulo kernel
-   `./containers.db`: Base de datos SQLite local
-   Scripts auxiliares para gestión de módulos y contenedores


