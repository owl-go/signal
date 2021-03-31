package kafka

import (
	"testing"
)

func TestSyncProducer(t *testing.T) {
	client, err := NewKafkaClient("localhost:9092")
	if err != nil {
		t.Error(err)
	}
	producer, err := NewSyncProducer(client)
	if err != nil {
		t.Error(err)
	}
	err = producer.Produce("waht", "i don't care")
	if err != nil {
		t.Error(err)
	}
	producer.Close()
	client.Close()
}
