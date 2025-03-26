package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/TommyLin81/kafka-demo/internal/entities"
	"github.com/TommyLin81/kafka-demo/internal/utils"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	clients              = make(map[*websocket.Conn]bool)
	broadcast            = make(chan entities.Message)
	producer             *kafka.Producer
	consumer             *kafka.Consumer
	chatMessagesTopic    = "chat-messages"
	filteredMessageTopic = "filtered-messages"
	bootstrapServers     string
)

func init() {
	bootstrapServers = os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if bootstrapServers == "" {
		bootstrapServers = "localhost:9092"
	}
}

func main() {
	var err error

	producer, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	fmt.Printf("create producer %v\n", producer)

	go utils.ListenProducerEvents(producer)

	consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"group.id":          "chat-group",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	fmt.Printf("create consumer %v\n", consumer)

	offset := kafka.OffsetBeginning
	offsets, err := consumer.Committed([]kafka.TopicPartition{{
		Topic:     &filteredMessageTopic,
		Partition: 0,
	}}, 1000)
	if err != nil {
		log.Println("get committed offsets failed:", err)
	}

	if len(offsets) > 0 {
		offset = offsets[0].Offset
	}

	err = consumer.Assign([]kafka.TopicPartition{{
		Topic:     &filteredMessageTopic,
		Partition: 0,
		Offset:    offset,
	}})
	if err != nil {
		log.Fatal(err)
	}

	go listenConsumerEvents()

	http.HandleFunc("/chat/1/connect", handleConnections)

	go handleMessages()

	fmt.Println("chat-server is running on port:80 ...")

	err = http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func listenConsumerEvents() {
	for {
		message, err := consumer.ReadMessage(-1)
		if err != nil {
			fmt.Printf("Consumer error: %v (%v)\n", err, message)
			continue
		}

		var chatMessage entities.Message
		err = json.Unmarshal(message.Value, &chatMessage)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
			continue
		}

		broadcast <- chatMessage
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket connect failed:", err)
		return
	}
	defer conn.Close()

	clients[conn] = true

	for {
		var msg entities.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("read message failed:", err)
			delete(clients, conn)
			break
		}

		msgBytes, _ := json.Marshal(msg)
		err = producer.Produce(
			&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &chatMessagesTopic,
					Partition: kafka.PartitionAny,
				},
				Value: msgBytes,
			},
			nil,
		)
		if err != nil {
			log.Println("produce message failed:", err)
		}
	}
}

func handleMessages() {
	for {
		msg := <-broadcast

		fmt.Println("handleMessages", msg)
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Println("send message failed:", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
