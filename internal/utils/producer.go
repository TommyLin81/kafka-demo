package utils

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func ListenProducerEvents(producer *kafka.Producer) {
	for e := range producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				log.Printf("topic: %v, delivery failed: %v\n", ev.TopicPartition.Topic, ev.TopicPartition.Error)
			} else {
				log.Printf("topic: %v, delivered: %v\n", ev.TopicPartition.Topic, ev.TopicPartition)
			}
		case kafka.Error:
			log.Printf("error: %v\n", ev)
		default:
			log.Printf("ignored event: %v\n", ev)
		}
	}
}
