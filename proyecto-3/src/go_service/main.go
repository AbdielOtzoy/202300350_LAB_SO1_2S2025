package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	pb "server_go/go_grpc_server/proto"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedWeatherTweetServiceServer
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
	// Conectar a RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return nil, nil, fmt.Errorf("error conectando a RabbitMQ: %v", err)
	}

	// Crear canal
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("error abriendo canal: %v", err)
	}

	// Declarar exchange (si no existe, lo crea)
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
		"weather.tweet.#",      // routing key (# = cualquier combinación)
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
	fmt.Println("   - Exchange: weather_exchange (topic)")
	fmt.Println("   - Queue: weather_tweets_queue")
	fmt.Println("   - Binding: weather.tweet.#")

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

// Método del servicio gRPC
func (s *server) SendTweet(ctx context.Context, req *pb.WeatherTweetRequest) (*pb.WeatherTweetResponse, error) {
	fmt.Printf("Go gRPC recibió: %+v\n", req)

	// Crear mensaje estructurado
	municipality := getMunicipalityName(req.Municipality)
	weather := getWeatherName(req.Weather)

	message := WeatherMessage{
		Municipality: municipality,
		Temperature:  req.Temperature,
		Humidity:     req.Humidity,
		Weather:      weather,
		Timestamp:    time.Now(),
		MessageID:    fmt.Sprintf("%s-%d", municipality, time.Now().Unix()),
	}

	// Serializar a JSON
	messageBody, err := json.Marshal(message)
	if err != nil {
		log.Printf("❌ Error serializando mensaje: %v", err)
		return &pb.WeatherTweetResponse{Status: "Error al procesar el mensaje"}, err
	}

	// Crear routing key: weather.tweet.{municipality}.{weather}
	routingKey := fmt.Sprintf("weather.tweet.%s.%s",
		strings.ToLower(municipality),
		strings.ToLower(weather))

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
			MessageId:    message.MessageID,
			Timestamp:    time.Now(),
			Headers: amqp.Table{
				"municipality": municipality,
				"weather":      weather,
				"temperature":  req.Temperature,
				"humidity":     req.Humidity,
			},
		},
	)

	if err != nil {
		log.Printf("❌ Error publicando en RabbitMQ: %v", err)
		return &pb.WeatherTweetResponse{Status: "Error al enviar a RabbitMQ"}, err
	}

	log.Printf("✅ Mensaje publicado en RabbitMQ con routing key: %s", routingKey)
	return &pb.WeatherTweetResponse{Status: "Tweet recibido y enviado a RabbitMQ correctamente ✅"}, nil
}

func main() {
	// Inicializar RabbitMQ
	conn, ch, err := initRabbitMQ()
	if err != nil {
		log.Fatalf("❌ Error inicializando RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	// Crear listener para gRPC
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Error escuchando: %v", err)
	}

	// Crear servidor gRPC
	grpcServer := grpc.NewServer()
	pb.RegisterWeatherTweetServiceServer(grpcServer, &server{
		rabbitConn:    conn,
		rabbitChannel: ch,
	})

	fmt.Println("🚀 Go gRPC server escuchando en :50051")
	fmt.Println("📡 Conectado a RabbitMQ")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("❌ Error en Serve: %v", err)
	}
}
