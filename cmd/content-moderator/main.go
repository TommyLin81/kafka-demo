package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

var producer *kafka.Producer
var consumer *kafka.Consumer
var chatMessagesTopic = "chat-messages"
var filteredMessageTopic = "filtered-messages"
var sensitiveWords = []string{"badword", "badword2"}

type Message struct {
	Username string `json:"username"`
	Message  string `json:"message"`
}

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
		Topic:     &chatMessagesTopic,
		Partition: 0,
	}}, 1000)

	if err != nil {
		log.Println("get committed offsets failed:", err)
	}

	if len(offsets) > 0 {
		offset = offsets[0].Offset
	}

	err = consumer.Assign([]kafka.TopicPartition{{
		Topic:     &chatMessagesTopic,
		Partition: 0,
		Offset:    offset,
	}})
	if err != nil {
		log.Fatal(err)
	}

	for {
		message, err := consumer.ReadMessage(-1)
		if err != nil {
			fmt.Printf("Consumer error: %v (%v)\n", err, message)
			continue
		}

		var msg Message
		err = json.Unmarshal(message.Value, &msg)
		if err != nil {
			fmt.Printf("Unmarshal error: %v\n", err)
			continue
		}

		fmt.Printf("Consumer received message: %v\n", msg)

		if containsSensitiveWords(msg.Message) {
			msg.Message = "🚨🚨🚨 [System] This message was blocked due to sensitive content. 🚨🚨🚨"
		}

		msgBytes, _ := json.Marshal(msg)
		err = producer.Produce(
			&kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &filteredMessageTopic,
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

func containsSensitiveWords(message string) bool {
	for _, word := range sensitiveWords {
		if strings.Contains(message, word) {
			return true
		}
	}

	return false
}
