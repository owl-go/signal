package kafka

import (
	"errors"
	"log"
	"strings"

	"github.com/Shopify/sarama"
)

type KafkaClient struct {
	sarama.Client
}

type SyncProducer struct {
	sarama.SyncProducer
}

func NewKafkaClient(url string) (*KafkaClient, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V0_10_2_0
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Consumer.Return.Errors = true
	client, err := sarama.NewClient(processUrlString(url), config)
	if err != nil {
		return nil, err
	}
	return &KafkaClient{
		Client: client,
	}, nil
}

func (k *KafkaClient) Close() error {
	return k.Client.Close()
}

func NewSyncProducer(client *KafkaClient) (*SyncProducer, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, err
	}
	return &SyncProducer{
		SyncProducer: producer,
	}, nil
}

func (s *SyncProducer) Produce(topic, message string) error {
	msg := &sarama.ProducerMessage{}
	msg.Topic = topic
	msg.Value = sarama.StringEncoder(message)
	pid, offset, err := s.SendMessage(msg)
	if err != nil {
		log.Printf("send message failed,err => %v", err)
		return err
	}
	log.Printf("sent success, pid:%v offset:%v", pid, offset)
	return nil
}

func (s *SyncProducer) Close() error {
	return s.SyncProducer.Close()
}

func processUrlString(url string) []string {
	urls := strings.Split(url, ",")
	for i, s := range urls {
		urls[i] = strings.TrimSpace(s)
	}
	return urls
}
