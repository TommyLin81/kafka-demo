package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/TommyLin81/kafka-demo/internal/entities"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan entities.Message)
var producer *kafka.Producer
var consumer *kafka.Consumer
var chatMessagesTopic = "chat-messages"
var filteredMessageTopic = "filtered-messages"

func main() {
	var err error

	producer, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	fmt.Printf("create producer %v\n", producer)

	go listenProducerEvents()

	consumer, err = kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
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
