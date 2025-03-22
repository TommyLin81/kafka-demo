package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var producer *kafka.Producer
var consumer *kafka.Consumer

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

func init() {
}

func main() {
	var err error

	producer, err = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "localhost:9092"})
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	fmt.Printf("create producer %v\n", producer)

	go listenProducerEvents()

	consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"group.id":          "chat-group",
		"auto.offset.reset": "latest",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	fmt.Printf("create consumer %v\n", consumer)

	err = consumer.Subscribe("chat-messages", nil)
	if err != nil {
		log.Fatal(err)
	}

	go listenConsumerEvents()

	http.HandleFunc("/chat/1/connect", handleConnections)

	go handleMessages()

	fmt.Println("chat-server is running on http://localhost:12345 ...")

	err = http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func listenProducerEvents() {
	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				fmt.Printf("Delivery failed: %v\n", ev.TopicPartition.Error)
			} else {
				fmt.Printf("Delivered message to %v\n", ev.TopicPartition)
			}
		case kafka.Error:
			fmt.Printf("Error: %v\n", ev)
		default:
			fmt.Printf("Ignored event: %v\n", ev)
		}
	}
}

func listenConsumerEvents() {
	for {
		message, err := consumer.ReadMessage(-1)
		if err != nil {
			fmt.Printf("Consumer error: %v (%v)\n", err, message)
		} else {
			var chatMessage Message
			err = json.Unmarshal(message.Value, &chatMessage)

			if err != nil {
				fmt.Printf("Unmarshal error: %v\n", err)
			} else {
				broadcast <- chatMessage
			}
		}
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
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("read message failed:", err)
			delete(clients, conn)
			break
		}

		topic := "chat-messages"
		msgBytes, _ := json.Marshal(msg)
		err = producer.Produce(
			&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &topic,
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
