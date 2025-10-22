package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "server_go/go_grpc_server/proto" // Asegúrate que la ruta sea correcta

	"github.com/IBM/sarama"

	"google.golang.org/grpc"
)

var (
	KAFKA_BROKER_ADDRESS = os.Getenv("KAFKA_BROKER_ADDRESS")
	KAFKA_TOPIC          = "weather_tweets"
	SERVER_PORT          = "50053"
)

// Estructura para el mensaje de Kafka (opcional, para serialización)
type KafkaMessage struct {
	Municipality string `json:"municipality"`
	Temperature  int32  `json:"temperature"`
	Humidity     int32  `json:"humidity"`
	Weather      string `json:"weather"`
	Timestamp    string `json:"timestamp"`
}

type kafkaPublisherServer struct {
	pb.UnimplementedWeatherPublisherServiceServer
	producer sarama.SyncProducer
}

// Implementación del nuevo método gRPC para Kafka
func (s *kafkaPublisherServer) PublishToKafka(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
	// 1. Convertir el mensaje gRPC a formato JSON/struct para Kafka
	municipality := getMunicipalityName(req.Municipality)
	weather := getWeatherName(req.Weather)

	message := KafkaMessage{
		Municipality: municipality,
		Temperature:  req.Temperature,
		Humidity:     req.Humidity,
		Weather:      weather,
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error al serializar mensaje para Kafka: %v", err)
		return &pb.WeatherTweetResponse{
			Status:    "Error interno: Fallo al serializar mensaje",
			MessageId: "",
		}, nil
	}

	// 2. Crear el mensaje para Kafka
	kafkaMsg := &sarama.ProducerMessage{
		Topic: KAFKA_TOPIC,
		Value: sarama.ByteEncoder(messageBytes),
		Key:   sarama.StringEncoder(municipality), // Usar el municipio como clave
	}

	// 3. Enviar a Kafka
	partition, offset, err := s.producer.SendMessage(kafkaMsg)
	if err != nil {
		log.Printf("❌ [SERVIDOR 3 - :%s] Error al publicar a Kafka: %v", SERVER_PORT, err)
		return &pb.WeatherTweetResponse{
			Status:    "Error: Fallo al publicar a Kafka",
			MessageId: "",
		}, err
	}

	messageID := fmt.Sprintf("%s-%d-%d", KAFKA_TOPIC, partition, offset)

	fmt.Printf("✅ [SERVIDOR 3 - :%s] Publicado a Kafka Topic: %s, Partition: %d, Offset: %d\n", SERVER_PORT, KAFKA_TOPIC, partition, offset)
	fmt.Printf("   Mensaje: %s\n", string(messageBytes))
	fmt.Println()

	return &pb.WeatherTweetResponse{
		Status:    "Mensaje publicado a Kafka exitosamente",
		MessageId: messageID,
	}, nil
}

// Funciones auxiliares (Municipality y Weather) - Copiadas del servidor 1
func getMunicipalityName(m pb.Municipalities) string {
	names := map[pb.Municipalities]string{
		pb.Municipalities_mixco:     "mixco",
		pb.Municipalities_guatemala: "guatemala",
		pb.Municipalities_amatitlan: "amatitlan",
		pb.Municipalities_chinautla: "chinautla",
	}
	if name, ok := names[m]; ok {
		return name
	}
	return "unknown"
}

func getWeatherName(w pb.Weathers) string {
	names := map[pb.Weathers]string{
		pb.Weathers_sunny:  "sunny",
		pb.Weathers_cloudy: "cloudy",
		pb.Weathers_rainy:  "rainy",
		pb.Weathers_foggy:  "foggy",
	}
	if name, ok := names[w]; ok {
		return name
	}
	return "unknown"
}

// Inicializa la conexión con Kafka
func initKafkaProducer() (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll // Esperar confirmación del líder y todos los réplicas
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true // Necesario para obtener el offset/partition

	// Crear el productor síncrono
	producer, err := sarama.NewSyncProducer([]string{os.Getenv("KAFKA_BROKER_ADDRESS")}, config)
	if err != nil {
		return nil, fmt.Errorf("fallo al crear el productor Sarama: %v", err)
	}
	fmt.Printf("✅ Conectado al broker de Kafka en %s\n", os.Getenv("KAFKA_BROKER_ADDRESS"))
	return producer, nil
}

func main() {
	fmt.Printf("🚀 Iniciando Servidor 3 (Kafka Publicador) en :%s...\n", SERVER_PORT)

	// Inicializar productor de Kafka
	producer, err := initKafkaProducer()
	if err != nil {
		log.Fatalf("❌ Error inicializando productor de Kafka: %v", err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			log.Printf("❌ Error cerrando productor de Kafka: %v", err)
		}
	}()

	// Crear listener para este servidor
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", SERVER_PORT))
	if err != nil {
		log.Fatalf("❌ Error escuchando en :%s: %v", SERVER_PORT, err)
	}

	// Crear servidor gRPC y registrar el servicio
	grpcServer := grpc.NewServer()
	pb.RegisterWeatherPublisherServiceServer(grpcServer, &kafkaPublisherServer{
		producer: producer,
	})

	// Manejo de señales para un cierre limpio
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n⚠️ Señal de terminación recibida. Deteniendo gRPC server...")
		grpcServer.GracefulStop()
	}()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ SERVIDOR 3 (Kafka Publicador) listo\n")
	fmt.Printf("    Puerto: :%s\n", SERVER_PORT)
	fmt.Printf("    Función: Recibir tweets gRPC y publicar a Kafka Topic: %s\n", KAFKA_TOPIC)
	fmt.Println("    Presiona CTRL+C para detener")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if err := grpcServer.Serve(lis); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Error en Serve: %v", err)
	}
	fmt.Println("Servidor detenido.")
}
