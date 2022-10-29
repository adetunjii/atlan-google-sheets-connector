package kafkahandler

import (
	"encoding/json"
	"testing"

	"github.com/adetunjii/google-sheets-connector/pkg/logger"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/stretchr/testify/require"
)

var (
	testPubTopic  = "googlesheet.test.topic"
	testSubTopics = []string{"googlesheet.test.topic"}
	testMessage   = map[string]interface{}{
		"title":  "google-sheets-connector",
		"author": "teej4y",
	}
)

func setupKafkaHandler() (*kafka.ConfigMap, logger.AppLogger) {

	zapLogger := logger.NewZapSugarLogger()
	logger := logger.NewLogger(zapLogger)

	config := &kafka.ConfigMap{
		"bootstrap.servers":  "pkc-n00kk.us-east-1.aws.confluent.cloud:9092",
		"security.protocol":  "SASL_SSL",
		"group.id":           "googlesheets",
		"sasl.mechanisms":    "PLAIN",
		"sasl.username":      "7CSJM4D3YDLCKGB2",
		"sasl.password":      "9IR9PwAmLyjh5iFzBaNfIxTIeVh2ogN46hSAm/CF37OCH/iSYwB74YPfMWa61iWD",
		"session.timeout.ms": 6000,
		"auto.offset.reset":  "earliest",
		"request.timeout.ms": 100000,
		"acks":               "all",
	}

	return config, logger
}

func newClient(t *testing.T) *KafkaHandler {
	config, logger := setupKafkaHandler()
	client := New(config, logger)

	require.NotNil(t, client)
	return client
}

func newProducer(t *testing.T) *kafka.Producer {
	kc := newClient(t)

	producer, err := kc.NewProducer()

	require.NoError(t, err)
	require.NotEmpty(t, producer)
	require.NotNil(t, producer)

	return producer
}

func TestPublish(t *testing.T) {
	kc := newClient(t)
	producer := newProducer(t)

	bytes, err := json.Marshal(testMessage)
	require.NoError(t, err)

	pErr := kc.Publish(producer, testPubTopic, bytes)
	require.NoError(t, pErr)

}

func newConsumer(t *testing.T) *kafka.Consumer {
	kc := newClient(t)
	consumer, err := kc.NewConsumer()
	require.NoError(t, err)
	require.NotEmpty(t, consumer)
	require.NotNil(t, consumer)

	return consumer
}

func TestSubscribe(t *testing.T) {
	kc := newClient(t)
	consumer := newConsumer(t)

	message, err := kc.Subscribe(consumer, testSubTopics)
	require.NoError(t, err)
	require.NotEmpty(t, message)
	require.NotNil(t, message)

	val := map[string]interface{}{}
	err = json.Unmarshal(message.Value, &val)
	require.NoError(t, err)
	require.NotEmpty(t, val)

	require.Equal(t, testMessage, val)
}
