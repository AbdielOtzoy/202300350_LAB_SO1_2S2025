package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	pb "server_go/go_grpc_server/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Definición de puertos
var (
	RABBITMQ_PUBLISHER_ADDRESS = os.Getenv("RABBITMQ_PUBLISHER_ADDRESS")
	KAFKA_PUBLISHER_ADDRESS    = os.Getenv("KAFKA_PUBLISHER_ADDRESS")
	RECEIVER_PORT              = os.Getenv("RECEIVER_PORT")
)

type publisherClients struct {
	rabbitClient pb.WeatherPublisherServiceClient
	kafkaClient  pb.WeatherPublisherServiceClient
}

type receptorServer struct {
	pb.UnimplementedWeatherTweetServiceServer
	clients *publisherClients
}

// Inicializar conexión con un servidor publicador (Servidor 2 y Servidor 3)
func initPublisherClient(address string, name string) (pb.WeatherPublisherServiceClient, *grpc.ClientConn, error) {
	// Opciones de conexión
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	}

	// Conectar
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("error conectando al servidor %s en %s: %v", name, address, err)
	}

	client := pb.NewWeatherPublisherServiceClient(conn)
	fmt.Printf("✅ Conectado al Servidor (%s) en :%s\n", name, address)

	return client, conn, nil
}

// Funciones auxiliares para logging (Municipality y Weather) (se mantienen igual)
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

// Implementación del método SendTweet
func (s *receptorServer) SendTweet(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
	municipality := getMunicipalityName(req.Municipality)
	weather := getWeatherName(req.Weather)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📥 [SERVIDOR 1 - :%s] Tweet recibido:\n", RECEIVER_PORT)
	fmt.Printf("    🏘️  Municipio: %s\n", municipality)
	fmt.Printf("    🌡️  Temperatura: %d°C\n", req.Temperature)
	fmt.Printf("    💧 Humedad: %d%%\n", req.Humidity)
	fmt.Printf("    ☁️  Clima: %s\n", weather)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Crear contexto con timeout para las llamadas a los publicadores
	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if rand.Intn(2) == 0 {
		// --- 1. Reenvío al Servidor Publicador de RabbitMQ (Puerto 5052) ---
		fmt.Printf("📤 [SERVIDOR 1] Reenviando a RabbitMQ Publicador (:%s)...", RABBITMQ_PUBLISHER_ADDRESS)
		// **IMPORTANTE**: Asumo que el Servidor 2 de RabbitMQ tiene el método PublishToRabbitMQ
		respRabbit, errRabbit := s.clients.rabbitClient.PublishToRabbitMQ(publishCtx, req)
		if errRabbit != nil {
			log.Printf("\n❌ [SERVIDOR 1] Error llamando al Publicador RabbitMQ (:%s): %v", RABBITMQ_PUBLISHER_ADDRESS, errRabbit)
		} else {
			fmt.Printf("\n✅ [SERVIDOR 1] Respuesta RabbitMQ Publicador: %s", respRabbit.Status)
			if respRabbit.MessageId != "" {
				fmt.Printf(" (ID: %s)", respRabbit.MessageId)
			}
			fmt.Println()
		}
		if errRabbit == nil {
			return respRabbit, nil
		}
		return &pb.WeatherTweetResponse{
			Status:    "Error: Fallo al enviar a AMBOS servidores publicadores",
			MessageId: "",
		}, fmt.Errorf("fallo al publicar a RabbitMQ (%v) y Kafka (%v)", errRabbit, nil)

	} else {
		// --- 2. Reenvío al NUEVO Servidor Publicador de Kafka (Puerto 5053) ---
		fmt.Printf("📤 [SERVIDOR 1] Reenviando al NUEVO Kafka Publicador (:%s)...", KAFKA_PUBLISHER_ADDRESS)
		// **IMPORTANTE**: Uso el método PublishToKafka que definiremos en el nuevo servidor.
		respKafka, errKafka := s.clients.kafkaClient.PublishToKafka(publishCtx, req)
		if errKafka != nil {
			log.Printf("\n❌ [SERVIDOR 1] Error llamando al Publicador Kafka (:%s): %v", KAFKA_PUBLISHER_ADDRESS, errKafka)
		} else {
			fmt.Printf("\n✅ [SERVIDOR 1] Respuesta Kafka Publicador: %s", respKafka.Status)
			if respKafka.MessageId != "" {
				fmt.Printf(" (ID: %s)", respKafka.MessageId)
			}
			fmt.Println()
		}
		if errKafka == nil {
			return respKafka, nil
		}
		return &pb.WeatherTweetResponse{
			Status:    "Error: Fallo al enviar a AMBOS servidores publicadores",
			MessageId: "",
		}, fmt.Errorf("fallo al publicar a RabbitMQ (%v) y Kafka (%v)", nil, errKafka)
	}

}

func main() {
	fmt.Printf("🚀 Iniciando Servidor 1 (Receptor) en :%s...\n", RECEIVER_PORT)

	// --- 1. Conectar al Publicador RabbitMQ (asumido en 5052) ---
	rabbitClient, rabbitConn, err := initPublisherClient(RABBITMQ_PUBLISHER_ADDRESS, "RabbitMQ")
	if err != nil {
		log.Fatalf("❌ Error inicializando cliente RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	kafkaClient, kafkaConn, err := initPublisherClient(KAFKA_PUBLISHER_ADDRESS, "Kafka")
	if err != nil {
		log.Fatalf("❌ Error inicializando cliente Kafka: %v", err)
	}
	defer kafkaConn.Close()

	// Agrupar los clientes
	pubClients := &publisherClients{
		rabbitClient: rabbitClient,
		kafkaClient:  kafkaClient,
	}

	// Crear listener para este servidor
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", RECEIVER_PORT))
	if err != nil {
		log.Fatalf("❌ Error escuchando en :%s: %v", RECEIVER_PORT, err)
	}

	// Crear servidor gRPC
	grpcServer := grpc.NewServer()
	pb.RegisterWeatherTweetServiceServer(grpcServer, &receptorServer{
		clients: pubClients,
	})

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("✅ SERVIDOR 1 (Receptor) listo\n")
	fmt.Printf("    Puerto: :%s\n", RECEIVER_PORT)
	fmt.Println("    Función: Recibir tweets y reenviar a RabbitMQ (5052) y Kafka (5053)")
	fmt.Println("    Presiona CTRL+C para detener")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Error en Serve: %v", err)
	}
}
