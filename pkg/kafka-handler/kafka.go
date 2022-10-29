package kafkahandler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/adetunjii/google-sheets-connector/pkg/logger"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

var (
	ErrInvalidAdminConfig     = errors.New("invalid kafka AdminClient configuration")
	ErrFailedTopicCreation    = errors.New("failed to create kafka topic")
	ErrTopicAlreadyExists     = errors.New("topic already exists")
	ErrFailedProducerCreation = errors.New("failed to create new producer")
	ErrFailedConsumerClose    = errors.New("failed to close consumer")
)

type KafkaHandler struct {
	config *kafka.ConfigMap
	topics map[string]struct{}
	logger logger.AppLogger
}

func New(config *kafka.ConfigMap, logger logger.AppLogger) *KafkaHandler {
	return &KafkaHandler{
		config: config,
		logger: logger,
		topics: make(map[string]struct{}),
	}
}

func (k *KafkaHandler) IsTopicExist(topic string) (bool, error) {
	isExist := false
	admin, err := createAdmin(k.config)
	if err != nil {
		k.logger.Error("failed to create kafka admin client  :: stacktrace :: ", err)
	}
	defer admin.Close()

	metadata, err := admin.GetMetadata(nil, true, int(5*time.Second))
	if err != nil {
		return isExist, fmt.Errorf("failed to get metadata :: stacktrace :: %v", err)
	}

	for k := range metadata.Topics {
		if k == topic {
			isExist = true
		}
	}

	return isExist, nil
}

func (k *KafkaHandler) CreateTopic(topic string) error {

	admin, err := createAdmin(k.config)
	if err != nil {
		k.logger.Error("failed to create kafka admin client :: stacktrace :: ", err)
	}
	defer admin.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxDuration, err := time.ParseDuration("60s")
	if err != nil {
		k.logger.Error("failed to create kafka topic :: stacktrace ::", err)
		return ErrFailedTopicCreation
	}

	results, err := admin.CreateTopics(
		ctx,
		[]kafka.TopicSpecification{
			{
				Topic:             topic,
				ReplicationFactor: 3,
				NumPartitions:     1,
			},
		},
		kafka.SetAdminOperationTimeout(maxDuration),
	)
	if err != nil {
		k.logger.Error("failed to create kafka topic :: stacktrace ::", err)
		return ErrFailedTopicCreation
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError && result.Error.Code() == kafka.ErrTopicAlreadyExists {
			return ErrTopicAlreadyExists
		}
	}

	k.topics[topic] = struct{}{}
	return nil
}

func (k *KafkaHandler) NewProducer() (*kafka.Producer, error) {
	p, err := kafka.NewProducer(k.config)
	if err != nil {
		k.logger.Error("failed to create a new producer :: stacktrace ::", err)
		return nil, ErrFailedProducerCreation
	}
	k.logger.Info("producer created succesfully")
	return p, nil
}

func (k *KafkaHandler) NewConsumer() (*kafka.Consumer, error) {
	c, err := kafka.NewConsumer(k.config)
	if err != nil {
		k.logger.Error("failed to create a new consumer :: stacktrace ::", err)
		return nil, ErrFailedProducerCreation
	}
	k.logger.Info("consumer created succesfully")
	return c, nil
}

func (k *KafkaHandler) Publish(producer *kafka.Producer, topic string, message []byte) error {

	isExist, err := k.IsTopicExist(topic)
	if err != nil {
		return err
	}

	// create new topic if it doesn't already exist
	if !isExist {
		if err := k.CreateTopic(topic); err != nil {
			return err
		}
	}

	m := newMessage(topic, message)

	if err := producer.Produce(m, nil); err != nil {
		return err
	}
	defer producer.Close()

	// asynchronous message delivery
	e := <-producer.Events()

	eventResponse := e.(*kafka.Message)
	if eventResponse.TopicPartition.Error != nil {
		k.logger.Error("message delivery failed :: stacktrace ::", eventResponse.TopicPartition.Error)
		return eventResponse.TopicPartition.Error
	} else {
		k.logger.Info(fmt.Sprintf("message delivered successfully to topic %s", topic))
	}

	return nil
}

func (k *KafkaHandler) Subscribe(consumer *kafka.Consumer, topics []string) (*kafka.Message, error) {

	var message *kafka.Message
	var err error

	err = consumer.SubscribeTopics(topics, nil)
	if err != nil {
		k.logger.Error("failed to subscribe to kafka :: stacktrace :: %v", err)
		return nil, err
	}
	defer consumer.Close()

	for {
		message, err = consumer.ReadMessage(10 * time.Second)
		if err == nil {
			return message, nil
		} else {
			k.logger.Error("failed to read message from kafka :: stacktrace ::", err)
			return nil, fmt.Errorf("failed to read message, %v", err)
		}
	}
}

func createAdmin(config *kafka.ConfigMap) (*kafka.AdminClient, error) {
	admin, err := kafka.NewAdminClient(config)
	if err != nil {
		if kErr, ok := err.(kafka.Error); ok {
			switch kErr.Code() {
			case kafka.ErrInvalidArg:
				return nil, ErrInvalidAdminConfig

			default:
				return nil, err
			}
		}
	}

	return admin, nil
}

func newMessage(topic string, message []byte) *kafka.Message {
	return &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          message,
	}
}
