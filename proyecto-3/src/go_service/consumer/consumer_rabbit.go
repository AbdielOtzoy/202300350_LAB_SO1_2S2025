package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

// Contexto global para operaciones con Redis
var ctx = context.Background()

// Estructura de mensaje recibido desde RabbitMQ
type WeatherMessage struct {
	Municipality string    `json:"municipality"`
	Temperature  int32     `json:"temperature"`
	Humidity     int32     `json:"humidity"`
	Weather      string    `json:"weather"`
	Timestamp    time.Time `json:"timestamp"`
	MessageID    string    `json:"message_id"`
}

func main() {
	// ============================
	// 🔗 Conexión a RabbitMQ
	// ============================
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		log.Fatalf("❌ Error conectando a RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("❌ Error abriendo canal: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"weather_tweets_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("❌ Error declarando queue: %v", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("❌ Error configurando QoS: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("❌ Error registrando consumidor: %v", err)
	}

	fmt.Println("🔄 Consumidor de RabbitMQ iniciado")
	fmt.Println("📬 Esperando mensajes en 'weather_tweets_queue'...\n")

	// ============================
	// 💾 Conexión a Valkey (Redis)
	// ============================
	addr := func() string {
		if v := os.Getenv("VALKEY_SERVICE_URL"); v != "" {
			return v
		}
		return "localhost:6379"
	}()

	var rdb *redis.Client
	if len(addr) >= 8 && addr[:8] == "redis://" {
		opt, err := redis.ParseURL(addr)
		if err != nil {
			log.Fatal("URL inválida en VALKEY_SERVICE_URL:", err)
		}
		rdb = redis.NewClient(opt)
	} else {
		rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: "",
			DB:       0,
		})
	}

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("❌ Error conectando a Valkey: %v", err)
	}
	fmt.Println("✅ Conectado a Valkey:", pong)

	// ============================
	// 🧠 Procesar mensajes
	// ============================
	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var weatherMsg WeatherMessage
			if err := json.Unmarshal(d.Body, &weatherMsg); err != nil {
				log.Printf("❌ Error deserializando mensaje: %v", err)
				d.Nack(false, false)
				continue
			}

			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Printf("📨 Nuevo mensaje recibido:\n")
			fmt.Printf("   🏘️  Municipio: %s\n", weatherMsg.Municipality)
			fmt.Printf("   🌡️  Temperatura: %d°C\n", weatherMsg.Temperature)
			fmt.Printf("   💧 Humedad: %d%%\n", weatherMsg.Humidity)
			fmt.Printf("   ☁️  Clima: %s\n", weatherMsg.Weather)
			fmt.Printf("   🕐 Timestamp: %s\n", weatherMsg.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			// Guardar en Valkey
			key := fmt.Sprintf("municipality:%s", weatherMsg.Municipality)
			err := rdb.HSet(ctx, key,
				"name", weatherMsg.Municipality,
				"temperature", weatherMsg.Temperature,
				"humidity", weatherMsg.Humidity,
				"weather", weatherMsg.Weather,
				"last_update", weatherMsg.Timestamp.Format("2006-01-02 15:04:05"),
			).Err()

			if err != nil {
				log.Printf("❌ Error guardando en Valkey: %v", err)
				d.Nack(false, true) // requeue
				continue
			}

			// Expira en 1 hora
			rdb.Expire(ctx, key, 1*time.Hour)

			fmt.Printf("💾 Datos almacenados en Valkey bajo la clave '%s'\n\n", key)

			// Confirmar mensaje
			d.Ack(false)
		}
	}()

	<-forever
}
