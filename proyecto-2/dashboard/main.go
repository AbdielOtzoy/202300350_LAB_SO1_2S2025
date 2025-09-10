package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Contenedor simulado
type Container struct {
	Name   string
	CPU    float64
	Memory float64
	Status string
}

type Proccess struct {
	PID          int
	Name         string
	Cmdline      string
	vsz          int
	rss          int
	Memory_Usage float64
	CPU_Usage    float64
	State        string
}

type Lecture struct {
	Totalram  int
	Freeram   int
	Procs     int
	Processes []Proccess
}

func main() {
	// ejecutar cron job al iniciar el daemon
	script := "/home/abdielo/dev/sopes/proyecto1/proyecto-2/bash/script.sh"

	if _, err := os.Stat(script); os.IsNotExist(err) {
		log.Fatalf("El script %s no existe", script)
	}

	if err := os.Chmod(script, 0755); err != nil {
		log.Printf("Advertencia: no se pudieron cambiar permisos del script: %v", err)
	}

	// Agregar cronjob (cada minuto)
	cronCommand := fmt.Sprintf("* * * * * %s", script)
	cmdCron := exec.Command("bash", "-c", fmt.Sprintf("(crontab -l 2>/dev/null; echo \"%s\") | crontab -", cronCommand))

	output, err := cmdCron.CombinedOutput()
	if err != nil {
		log.Fatalf("Error agregando cronjob: %v, output: %s", err, string(output))
	}

	log.Printf("Cronjob creado correctamente: %s", cronCommand)

	// Ejecutar comando docker antes de todo
	cmd := exec.Command("docker", "start", "5fdbd83f343c")
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Error al ejecutar docker start: %v\nSalida: %s", err, string(output))
	}
	log.Printf("Contenedor iniciado con éxito. Salida: %s", string(output))

	// Cargar módulo del kernel de la información del sistema
	loadCmd := exec.Command("/home/abdielo/dev/sopes/proyecto1/proyecto-2/bash/mod_manager.sh", "load")
	output, err = loadCmd.CombinedOutput()
	if err != nil {
		log.Printf("Error loading modules: %v, Output: %s", err, output)
	}
	log.Printf("Modules loaded successfully: %s", output)
	// Inicializar aleatoriedad
	rand.Seed(time.Now().UnixNano())

	// Conectar a SQLite (crea el archivo si no existe)
	db, err := sql.Open("sqlite3", "./containers.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Crear tabla si no existe
	createTable := `
	CREATE TABLE IF NOT EXISTS containers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		cpu REAL,
		memory REAL,
		status TEXT,
		created_at INTEGER
	);`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal("Error creando tabla:", err)
	}

	createContinfoTable := `
	CREATE TABLE IF NOT EXISTS continfo (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pid INTEGER,
		name TEXT,
		cmdline TEXT,
		vsz INTEGER,
		rss INTEGER,
		memory_usage REAL,
		cpu_usage REAL,
		state TEXT
	);`
	_, err = db.Exec(createContinfoTable)
	if err != nil {
		log.Fatal("Error creando tabla continfo:", err)
	}

	createSysinfoTable := `
	CREATE TABLE IF NOT EXISTS sysinfo (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		totalram INTEGER,
		freeram INTEGER,
		usedram INTEGER,
		containersDeleted INTEGER,
		procs INTEGER
	);`
	_, err = db.Exec(createSysinfoTable)
	if err != nil {
		log.Fatal("Error creando tabla sysinfo:", err)
	}

	// containersDeleted := 0 // count de contenedores eliminados

	// Canal para capturar señales (detener daemon)
	// 'sigs' es un canal que se utiliza para recibir señales del sistema operativo.
	// En este caso, se capturan señales SIGINT (Ctrl+C) y SIGTERM (terminación del proceso).
	// Estas señales permiten detener el daemon de manera controlada.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Daemon iniciado. Generando datos cada 44 segundos...")

	// 'ticker' es un temporizador que genera eventos periódicos cada 44 segundos.
	// Estos eventos se utilizan para ejecutar tareas recurrentes, como generar e insertar contenedores aleatorios.
	ticker := time.NewTicker(20 * time.Second)
	// 'defer ticker.Stop()' asegura que el temporizador se detenga correctamente cuando el programa termine.
	defer ticker.Stop()

	// Este bucle infinito permite que el daemon esté en ejecución constante,
	// esperando eventos del ticker o señales del sistema operativo.
loop:
	// El for select es una construcción que permite esperar en múltiples canales.
	for {
		select {
		case <-ticker.C:
			// Este caso se ejecuta cada vez que el ticker genera un evento (cada 44 segundos).
			// Genera contenedores aleatorios y los inserta en la base de datos.
			containers := generateRandomContainers()
			for _, c := range containers {
				_, err := db.Exec("INSERT INTO containers (name, cpu, memory, status, created_at) VALUES (?, ?, ?, ?, ?)",
					c.Name, c.CPU, c.Memory, c.Status, time.Now().UnixMilli())
				if err != nil {
					// Si ocurre un error al insertar, se registra en los logs.
					log.Println("Error insertando:", err)
				} else {
					// Si la inserción es exitosa, se registra el contenedor insertado en los logs.
					log.Printf("Insertado contenedor random: %s (CPU: %.2f, MEM: %.2f, Estado: %s)\n",
						c.Name, c.CPU, c.Memory, c.Status)
				}
			}
			// lecture := readContinfo()
			// log.Printf("Lectura de continfo: %+v\n", lecture)
			// // por cada proceso en la lectura, insertarlo en la base de datos
			// for _, p := range lecture.Processes {
			// 	_, err := db.Exec("INSERT INTO continfo (pid, name, cmdline, vsz, rss, memory_usage, cpu_usage, state) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			// 		p.PID, p.Name, p.Cmdline, p.vsz, p.rss, p.Memory_Usage, p.CPU_Usage, p.State)
			// 	if err != nil {
			// 		log.Println("Error insertando en continfo:", err)
			// 	} else {
			// 		log.Printf("Insertado en continfo: %s (PID: %d, Estado: %s)\n",
			// 			p.Name, p.PID, p.State)
			// 	}
			// }

			// syslecture := readSysinfo()
			// usedram := syslecture.Totalram - syslecture.Freeram
			// log.Printf("Lectura de sysinfo")
			// _, err := db.Exec("INSERT INTO sysinfo (totalram, freeram, usedram, containersDeleted, procs) VALUES (?, ?, ?, ?, ?)",
			// 	syslecture.Totalram, syslecture.Freeram, usedram, containersDeleted, syslecture.Procs)
			// if err != nil {
			// 	log.Println("Error insertando en sysinfo:", err)
			// } else {
			// 	log.Printf("Insertado en sysinfo: TotalRAM: %d, FreeRAM: %d, UsedRAM: %d, ContainersDeleted: %d, Procs: %d\n",
			// 		syslecture.Totalram, syslecture.Freeram, usedram, containersDeleted, syslecture.Procs)
			// }
		case <-sigs:
			cmd := exec.Command("crontab", "-r")
			if err := cmd.Run(); err != nil {
				log.Printf("Error eliminando cronjobs: %v", err)
			} else {
				log.Println("Todos los cronjobs eliminados correctamente")
			}

			// unload módulo del kernel al detener el daemon
			unloadCmd := exec.Command("/home/abdielo/dev/sopes/proyecto1/proyecto-2/bash/mod_manager.sh", "unload")
			unloadOut, err := unloadCmd.CombinedOutput()
			if err != nil {
				log.Printf("Error unloading modules: %v\nSalida: %s", err, string(unloadOut))
			} else {
				log.Printf("Modules unloaded successfully. Salida: %s", string(unloadOut))
			}

			// para el contenedor docker al detener el daemon
			stopCmd := exec.Command("docker", "stop", "5fdbd83f343c")
			stopOut, err := stopCmd.CombinedOutput()
			if err != nil {
				log.Printf("Error al detener contenedor: %v\nSalida: %s", err, string(stopOut))
			} else {
				log.Printf("Contenedor detenido con éxito. Salida: %s", string(stopOut))
			}

			// Este caso se ejecuta cuando se recibe una señal de interrupción o terminación.
			// Detiene el daemon y sale del bucle.
			log.Println("Daemon detenido.")
			break loop
		}
	}
}

// Genera hasta 4 contenedores con datos random
func generateRandomContainers() []Container {
	names := []string{"nginx", "redis", "mysql", "golang-app", "nodejs-app", "python-app", "java-app", "ruby-app", "postgres", "mongodb", "ubuntu", "alpine"}
	statuses := []string{"running", "stopped", "paused", "restarting", "dead", "zombie"}

	n := rand.Intn(4) + 1 // mínimo 1, máximo 4
	var containers []Container
	for i := 0; i < n; i++ {
		containers = append(containers, Container{
			Name:   names[rand.Intn(len(names))],
			CPU:    rand.Float64() * 100, // 0-100%
			Memory: rand.Float64() * 512, // hasta 1GB
			Status: statuses[rand.Intn(len(statuses))],
		})
	}
	return containers
}

func readSysinfo() Lecture {
	cmd := exec.Command("cat", "/proc/sysinfo_so1_202300350")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("Error reading sysinfo:", err)
	}

	var sysinfo Lecture
	if err := json.Unmarshal(output, &sysinfo); err != nil {
		log.Fatal("Error unmarshalling sysinfo:", err)
	}

	return sysinfo
}

func readContinfo() Lecture {
	cmd := exec.Command("cat", "/proc/continfo_so1_202300350")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("Error reading continfo:", err)
	}

	var continfo Lecture
	if err := json.Unmarshal(output, &continfo); err != nil {
		log.Fatal("Error unmarshalling continfo:", err)
	}

	return continfo
}
