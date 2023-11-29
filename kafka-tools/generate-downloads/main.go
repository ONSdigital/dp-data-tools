package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	cantabularEvent "github.com/ONSdigital/dp-cantabular-filter-flex-api/event"
	cantabularEventSchema "github.com/ONSdigital/dp-cantabular-filter-flex-api/schema"
	cmdEvent "github.com/ONSdigital/dp-dataset-api/download"
	cmdEventSchema "github.com/ONSdigital/dp-dataset-api/schema"
	kafka "github.com/ONSdigital/dp-kafka/v3"
	"github.com/ONSdigital/dp-kafka/v3/avro"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/kelseyhightower/envconfig"
)

const KafkaTLSProtocolFlag = "TLS"

type Config struct {
	DatasetID   string        `envconfig:"DATASET_ID"`
	InstanceID  string        `envconfig:"INSTANCE_ID"`
	Edition     string        `envconfig:"EDITION"`
	Version     string        `envconfig:"VERSION"`
	Timeout     time.Duration `envconfig:"TIMEOUT"`
	Service     string        `envconfig:"SERVICE"`
	KafkaConfig KafkaConfig
}

type KafkaConfig struct {
	Brokers                   []string `envconfig:"KAFKA_ADDR"`
	Version                   string   `envconfig:"KAFKA_VERSION"`
	MaxBytes                  int      `envconfig:"KAFKA_MAX_BYTES"`
	ProducerMinBrokersHealthy int      `envconfig:"KAFKA_PRODUCER_MIN_BROKERS_HEALTHY"`
	SecClientKey              string   `envconfig:"KAFKA_SEC_CLIENT_KEY"        json:"-"`
	SecClientCert             string   `envconfig:"KAFKA_SEC_CLIENT_CERT"`
	SecCACerts                string   `envconfig:"KAFKA_SEC_CA_CERTS"`
	SecSkipVerify             bool     `envconfig:"KAFKA_SEC_SKIP_VERIFY"`
	Topic                     string   `envconfig:"KAFA_PRODUCER_TOPIC"`
	SecProtocol               string   `envconfig:"KAFKA_SEC_PROTO"`
}

var defaultCfg = Config{
	KafkaConfig: KafkaConfig{
		Brokers:                   []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"},
		Version:                   "1.0.2",
		Topic:                     "cantabular-export-start",
		ProducerMinBrokersHealthy: 2,
	},
	Timeout: 30 * time.Second,
	Edition: "2021",
	Service: "cantabular",
}

func main() {
	cfg := getConfig()
	ctx := context.Background()
	log.Info(ctx, "Config", log.Data{"config": cfg})

	pConfig := &kafka.ProducerConfig{
		KafkaVersion:      &cfg.KafkaConfig.Version,
		MaxMessageBytes:   &cfg.KafkaConfig.MaxBytes,
		BrokerAddrs:       cfg.KafkaConfig.Brokers,
		Topic:             cfg.KafkaConfig.Topic,
		MinBrokersHealthy: &cfg.KafkaConfig.ProducerMinBrokersHealthy,
	}

	if cfg.KafkaConfig.SecProtocol == KafkaTLSProtocolFlag {
		log.Info(ctx, "Producer getting TLS")
		pConfig.SecurityConfig = kafka.GetSecurityConfig(
			cfg.KafkaConfig.SecCACerts,
			cfg.KafkaConfig.SecClientCert,
			cfg.KafkaConfig.SecClientKey,
			cfg.KafkaConfig.SecSkipVerify,
		)
	}

	producer, err := kafka.NewProducer(ctx, pConfig)
	if err != nil {
		log.Error(ctx, "NewProducer failed", err)
		return
	}

	producer.LogErrors(ctx)

	log.Info(ctx, "message send...")

	event, schema := getEventAndSchema(*cfg)

	if err := producer.Send(schema, event); err != nil {
		log.Error(ctx, "error sending 'export_start' event", err)
		return
	}

	log.Info(ctx, "closing producer...")
	if err := producer.Close(ctx); err != nil {
		log.Error(ctx, "close failed", err)
	}
	log.Info(ctx, "producer closed")
}

func getConfig() (cfg *Config) {
	cfg = &Config{}
	*cfg = defaultCfg

	if err := envconfig.Process("", cfg); err != nil {
		panic(err)
	}
	if cfg.DatasetID == "" {
		panic("no dataset id")
	}
	if cfg.InstanceID == "" {
		panic("no instance id")
	}
	if cfg.Edition == "" {
		panic("no edition")
	}
	if cfg.Version == "" {
		panic("no version")
	}
	if cfg.Service == "" {
		panic("no service")
	}
	if cfg.KafkaConfig.SecProtocol != "" {
		if cfg.KafkaConfig.SecProtocol != KafkaTLSProtocolFlag {
			panic("bad kafka sec proto - expected: " + KafkaTLSProtocolFlag)
		}
		cfg.KafkaConfig.SecClientCert = strings.Replace(cfg.KafkaConfig.SecClientCert, "\\n", "\n", -1)
		cfg.KafkaConfig.SecClientKey = strings.Replace(cfg.KafkaConfig.SecClientKey, "\\n", "\n", -1)
	}
	return
}

func (config Config) String() string {
	jsonStr, _ := json.Marshal(config)
	return string(jsonStr)
}

func getEventAndSchema(config Config) (interface{}, *avro.Schema) {
	if config.Service == "cantabular" {
		return getCantabularEventAndSchema(config)
	} else {
		return getCMDEventAndSchema(config)
	}
}

func getCantabularEventAndSchema(config Config) (interface{}, *avro.Schema) {
	e := cantabularEvent.ExportStart{
		InstanceID: config.InstanceID,
		DatasetID:  config.DatasetID,
		Edition:    config.Edition,
		Version:    config.Version,
	}
	return e, cantabularEventSchema.ExportStart
}

func getCMDEventAndSchema(config Config) (interface{}, *avro.Schema) {
	e := cmdEvent.GenerateDownloads{
		InstanceID: config.InstanceID,
		DatasetID:  config.DatasetID,
		Edition:    config.Edition,
		Version:    config.Version,
	}
	return e, cmdEventSchema.GenerateCMDDownloadsEvent
}
