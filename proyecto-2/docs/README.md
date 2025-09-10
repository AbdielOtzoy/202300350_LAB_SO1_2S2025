# Manuel Técnico

Instrucciones para compilar un módulo de kernel en Linux.
Quitar el módulo actual:

```bash
sudo rmmod sysinfo
```


Revisar que ya no esté:
```bash
lsmod | grep sysinfo
```


Limpiar y recompilar:
```bash
make clean
make
```


Cargar el módulo nuevo:
```bash
sudo insmod sysinfo.ko
```


Verificar en /proc:
```bash
cat /proc/sysinfo_so1_202300350

```

Instalar Daemon systemd

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