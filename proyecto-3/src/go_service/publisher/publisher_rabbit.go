package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	pb "server_go/go_grpc_server/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

type publisherServer struct {
	pb.UnimplementedWeatherPublisherServiceServer
	rabbitConn    *amqp.Connection
	rabbitChannel *amqp.Channel
}

// Estructura para el mensaje que se enviará a RabbitMQ
type WeatherMessage struct {
	Municipality string    `json:"municipality"`
	Temperature  int32     `json:"temperature"`
	Humidity     int32     `json:"humidity"`
	Weather      string    `json:"weather"`
	Timestamp    time.Time `json:"timestamp"`
	MessageID    string    `json:"message_id"`
}

// Inicializar conexión con RabbitMQ
func initRabbitMQ() (*amqp.Connection, *amqp.Channel, error) {
	fmt.Println("🔄 Conectando a RabbitMQ...")

	// Conectar a RabbitMQ con reintentos
	var conn *amqp.Connection
	var err error
	maxRetries := 5

	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(os.Getenv("RABBITMQ_URL"))
		if err == nil {
			break
		}
		fmt.Printf("   Reintento %d/%d...\n", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("error conectando a RabbitMQ después de %d intentos: %v", maxRetries, err)
	}

	// Crear canal
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("error abriendo canal: %v", err)
	}

	// Habilitar modo de confirmación
	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, fmt.Errorf("error habilitando confirmaciones: %v", err)
	}

	// Declarar exchange
	err = ch.ExchangeDeclare(
		"weather_exchange", // nombre
		"topic",            // tipo
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, fmt.Errorf("error declarando exchange: %v", err)
	}

	// Declarar queue principal
	_, err = ch.QueueDeclare(
		"weather_tweets_queue", // nombre
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, fmt.Errorf("error declarando queue: %v", err)
	}

	// Binding del queue al exchange
	err = ch.QueueBind(
		"weather_tweets_queue", // queue name
		"weather.tweet.#",      // routing key
		"weather_exchange",     // exchange
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, fmt.Errorf("error en binding: %v", err)
	}

	fmt.Println("✅ RabbitMQ configurado correctamente")
	fmt.Println("   Exchange: weather_exchange (topic)")
	fmt.Println("   Queue: weather_tweets_queue")
	fmt.Println("   Binding: weather.tweet.#")

	return conn, ch, nil
}

// Convertir enum a string
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

// Implementación del método PublishToRabbitMQ
func (s *publisherServer) PublishToRabbitMQ(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
	municipality := getMunicipalityName(req.Municipality)
	weather := getWeatherName(req.Weather)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📨 [SERVIDOR 2] Mensaje recibido del Servidor 1:\n")
	fmt.Printf("   🏘️  Municipio: %s\n", municipality)
	fmt.Printf("   🌡️  Temperatura: %d°C\n", req.Temperature)
	fmt.Printf("   💧 Humedad: %d%%\n", req.Humidity)
	fmt.Printf("   ☁️  Clima: %s\n", weather)

	// Crear mensaje estructurado
	messageID := fmt.Sprintf("%s-%d", municipality, time.Now().Unix())
	message := WeatherMessage{
		Municipality: municipality,
		Temperature:  req.Temperature,
		Humidity:     req.Humidity,
		Weather:      weather,
		Timestamp:    time.Now(),
		MessageID:    messageID,
	}

	// Serializar a JSON
	messageBody, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ [SERVIDOR 2] Error serializando mensaje: %v", err)
		return &pb.WeatherTweetResponse{
			Status:    "Error al procesar el mensaje",
			MessageId: "",
		}, err
	}

	// Crear routing key
	routingKey := fmt.Sprintf("weather.tweet.%s.%s",
		strings.ToLower(municipality),
		strings.ToLower(weather))

	fmt.Printf("🔑 [SERVIDOR 2] Routing key: %s\n", routingKey)
	fmt.Println("📤 [SERVIDOR 2] Publicando en RabbitMQ...")

	// Publicar mensaje en RabbitMQ
	err = s.rabbitChannel.PublishWithContext(
		ctx,
		"weather_exchange", // exchange
		routingKey,         // routing key
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         messageBody,
			DeliveryMode: amqp.Persistent, // Mensaje persistente
			MessageId:    messageID,
			Timestamp:    time.Now(),
			Headers: amqp.Table{
				"municipality": municipality,
				"weather":      weather,
				"temperature":  req.Temperature,
				"humidity":     req.Humidity,
				"source":       "grpc_server_2",
			},
		},
	)

	if err != nil {
		log.Printf("❌ [SERVIDOR 2] Error publicando en RabbitMQ: %v", err)
		return &pb.WeatherTweetResponse{
			Status:    "Error al enviar a RabbitMQ",
			MessageId: "",
		}, err
	}

	fmt.Printf("✅ [SERVIDOR 2] Mensaje publicado exitosamente\n")
	fmt.Printf("   ID: %s\n", messageID)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return &pb.WeatherTweetResponse{
		Status:    "Tweet enviado a RabbitMQ correctamente ✅",
		MessageId: messageID,
	}, nil
}

func main() {
	fmt.Println("🚀 Iniciando Servidor 2 (Publicador RabbitMQ)...")

	// Inicializar RabbitMQ
	conn, ch, err := initRabbitMQ()
	if err != nil {
		log.Fatalf("❌ Error inicializando RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	// Manejar cierre inesperado de RabbitMQ
	notifyClose := make(chan *amqp.Error)
	ch.NotifyClose(notifyClose)

	go func() {
		err := <-notifyClose
		if err != nil {
			log.Printf("⚠️  Conexión con RabbitMQ cerrada: %v", err)
		}
	}()

	// Crear listener para este servidor
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("❌ Error escuchando en :50052: %v", err)
	}

	// Crear servidor gRPC
	grpcServer := grpc.NewServer()
	pb.RegisterWeatherPublisherServiceServer(grpcServer, &publisherServer{
		rabbitConn:    conn,
		rabbitChannel: ch,
	})

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ SERVIDOR 2 (Publicador) listo")
	fmt.Println("   Puerto: :50052")
	fmt.Println("   Función: Publicar mensajes en RabbitMQ")
	fmt.Println("   Presiona CTRL+C para detener")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Error en Serve: %v", err)
	}
}
