package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/valkey-io/valkey-go"
)

// Estructura del mensaje
type WeatherMessage struct {
	Municipality string `json:"municipality"`
	Temperature  int32  `json:"temperature"`
	Humidity     int32  `json:"humidity"`
	Weather      string `json:"weather"`
	Timestamp    string `json:"timestamp"`
}

var (
	KAFKA_BROKER = os.Getenv("KAFKA_BROKER_ADDRESS")
	KAFKA_TOPIC  = "weather_tweets"
	GROUP_ID     = "weather_consumers"
	VALKEY_ADDR  = os.Getenv("VALKEY_ADDRESS")
)

// Handler del consumidor
type ConsumerHandler struct {
	valkeyClient valkey.Client
}

func (h ConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error {
	fmt.Println("✅ Consumidor conectado a Kafka y listo para recibir mensajes...")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	return nil
}

func (h ConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	fmt.Println("⚠️ Cerrando consumidor...")
	h.valkeyClient.Close()
	return nil
}

func (h ConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	ctx := context.Background()

	for message := range claim.Messages() {
		var weatherMsg WeatherMessage
		err := json.Unmarshal(message.Value, &weatherMsg)
		if err != nil {
			log.Printf("❌ Error deserializando mensaje: %v", err)
			continue
		}

		// Mostrar el mensaje recibido
		fmt.Printf("📨 Nuevo mensaje recibido desde Kafka:\n")
		fmt.Printf("   🏘️  Municipio: %s\n", weatherMsg.Municipality)
		fmt.Printf("   🌡️  Temperatura: %d°C\n", weatherMsg.Temperature)
		fmt.Printf("   💧 Humedad: %d%%\n", weatherMsg.Humidity)
		fmt.Printf("   ☁️  Clima: %s\n", weatherMsg.Weather)
		fmt.Printf("   🕐 Timestamp: %s\n", weatherMsg.Timestamp)
		fmt.Printf("   📦 Topic: %s | Partition: %d | Offset: %d\n",
			message.Topic, message.Partition, message.Offset)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		// 🔥 Guardar en Valkey
		key := fmt.Sprintf("weather:%s:%s", weatherMsg.Municipality, weatherMsg.Timestamp)
		value, _ := json.Marshal(weatherMsg)

		err = h.valkeyClient.Do(ctx, h.valkeyClient.B().Set().Key(key).Value(string(value)).Build()).Error()
		if err != nil {
			log.Printf("❌ Error guardando en Valkey: %v", err)
		} else {
			fmt.Printf("💾 Guardado en Valkey con clave: %s\n", key)
		}

		// Confirmar que el mensaje fue procesado
		session.MarkMessage(message, "")
	}

	return nil
}

func main() {
	fmt.Println("🚀 Iniciando consumidor de Kafka...")
	fmt.Printf("📡 Broker: %s | Topic: %s | Grupo: %s\n", KAFKA_BROKER, KAFKA_TOPIC, GROUP_ID)

	// Crear conexión a Valkey
	valkeyClient, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{VALKEY_ADDR},
	})
	if err != nil {
		log.Fatalf("❌ Error conectando a Valkey: %v", err)
	}
	defer valkeyClient.Close()

	// Configuración de Sarama
	config := sarama.NewConfig()
	config.Version = sarama.V2_5_0_0
	config.Consumer.Return.Errors = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin

	consumerGroup, err := sarama.NewConsumerGroup([]string{KAFKA_BROKER}, GROUP_ID, config)
	if err != nil {
		log.Fatalf("❌ Error creando grupo de consumidores: %v", err)
	}
	defer consumerGroup.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	handler := ConsumerHandler{valkeyClient: valkeyClient}

	go func() {
		for {
			if err := consumerGroup.Consume(ctx, []string{KAFKA_TOPIC}, handler); err != nil {
				log.Printf("❌ Error en el consumo: %v", err)
				time.Sleep(2 * time.Second)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	<-ctx.Done()
	fmt.Println("🛑 Consumidor detenido correctamente.")
}
