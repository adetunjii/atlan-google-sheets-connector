package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/adetunjii/google-sheets-connector/internal/handler/httphandler"
	"github.com/adetunjii/google-sheets-connector/internal/model"
	"github.com/adetunjii/google-sheets-connector/internal/services/google"
	kafkahandler "github.com/adetunjii/google-sheets-connector/pkg/kafka-handler"
	"github.com/adetunjii/google-sheets-connector/pkg/logger"
	"github.com/adetunjii/google-sheets-connector/pkg/monitoring"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

func main() {

	if err := setupViper(); err != nil {
		log.Fatal(err)
	}

	port := viper.GetString("PORT")

	google_client_id := viper.GetString("GOOGLE_CLIENT_ID")
	google_client_secret := viper.GetString("GOOGLE_CLIENT_SECRET")
	google_scopes := viper.GetStringSlice("GOOGLE_SCOPES")
	google_callback_url := viper.GetString("GOOGLE_CALLBACK_URL")

	sg := logger.NewZapSugarLogger()
	logger := logger.NewLogger(sg)

	googleClient := google.NewGoogleClient(
		google_client_id,
		google_client_secret,
		google_scopes,
		google_callback_url,
		logger,
	)

	// setup metrics and monitoring
	metrics := monitoring.NewMetricsWrapper(
		monitoring.ServiceName("google-sheets-connector"),
		monitoring.ServiceMetricsLabelPrefix("gsc"),
	)

	// setup kafka
	kafkahandler := setupKafka(logger)
	kafkaTopics := viper.GetStringSlice("KAFKA_TOPICS")
	kafkaConsumer, err := kafkahandler.NewConsumer()
	if err != nil {
		logger.Error("failed to create kafka consumer :: stacktrace :: ", err)
	}

	go func() {
		km := model.GoogleSheetKafkaMessage{}
		message, err := kafkahandler.Subscribe(kafkaConsumer, kafkaTopics)
		if err != nil {
			logger.Error("topic failed subscription failed :: stacktrace :: ", err)
		}

		if err := json.Unmarshal(message.Value, &km); err != nil {
			logger.Error("failed to parse message from broker :: stacktrace ::", err)
		}

		googleSheetClient := google.NewGoogleSheetClient(googleClient, km.Token, logger)

		writeErr := googleSheetClient.WriteToSheet(km.SpreadSheetID, &km.Questionnaire)
		if writeErr != nil {
			logger.Error("failed to write message to google sheets :: stacktrace ::", writeErr)
		}
	}()

	router := mux.NewRouter()
	httpHandler := httphandler.New(googleClient, logger)

	router.Use(metrics.MetricsMiddleware)
	router.Path("/metrics").Handler(promhttp.Handler())
	router.Path("/api/google-sheets/integrate").HandlerFunc(httpHandler.OauthGoogle)
	router.Path("/api/google-sheets/integrate/callback").HandlerFunc(httpHandler.OauthGoogleCallback)
	router.Path("/api/google-sheets/create").HandlerFunc(httpHandler.CreateGoogleSheet).Methods(http.MethodPost)

	s := http.Server{
		Addr:        fmt.Sprintf(":%v", port),
		IdleTimeout: 120 * time.Second,
		Handler:     router,
	}

	logger.Info(fmt.Sprintf("Server is running on port %v 🚀🚀", port))
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			log.Fatal(err)
		}
	}()

	// attempt graceful shutdown within 30 seconds before closing
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	sig := <-sigChan
	log.Println("recieved graceful shutdown", sig)

	tc, _ := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(tc)
}

func setupViper() error {
	viper.SetConfigName("app")
	viper.AddConfigPath(".")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			viper.AutomaticEnv()
		} else {
			return fmt.Errorf("cannot read config: %v", err)
		}
	}

	return nil
}

func setupKafka(logger logger.AppLogger) *kafkahandler.KafkaHandler {
	kafka_brokers := viper.GetString("KAFKA_BROKERS")
	kafka_username := viper.GetString("KAFKA_USERNAME")
	kafka_password := viper.GetString("KAFKA_PASSWORD")
	kafka_group_id := viper.GetString("SERVICE_ID")

	config := &kafka.ConfigMap{
		"bootstrap.servers":  kafka_brokers,
		"group.id":           kafka_group_id,
		"security.protocol":  "SASL_SSL",
		"sasl.mechanisms":    "PLAIN",
		"sasl.username":      kafka_username,
		"sasl.password":      kafka_password,
		"session.timeout.ms": 6000,
		"auto.offset.reset":  "earliest",
		"request.timeout.ms": 100000,
		"acks":               "all",
	}

	kHandler := kafkahandler.New(config, logger)
	return kHandler
}
