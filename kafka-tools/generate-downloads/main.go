package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/ONSdigital/dp-dataset-api/download"
	adapter "github.com/ONSdigital/dp-dataset-api/kafka"
	"github.com/ONSdigital/dp-dataset-api/schema"
	kafka "github.com/ONSdigital/dp-kafka/v2"
	"github.com/ONSdigital/log.go/v2/log"
)

const KafkaTLSProtocolFlag = "TLS"

type Config struct {
	DatasetID   string        `envconfig:"DATASET_ID"`
	InstanceID  string        `envconfig:"INSTANCE_ID"`
	Edition     string        `envconfig:"EDITION"`
	Version     string        `envconfig:"VERSION"`
	Timeout     time.Duration `envconfig:"TIMEOUT"`
	KafkaConfig KafkaConfig
}

type KafkaConfig struct {
	Brokers                []string `envconfig:"KAFKA_ADDR"`
	Version                string   `envconfig:"KAFKA_VERSION"`
	MaxBytes               int      `envconfig:"KAFKA_MAX_BYTES"`
	SecProtocol            string   `envconfig:"KAFKA_SEC_PROTO"`
	SecClientKey           string   `envconfig:"KAFKA_SEC_CLIENT_KEY"        json:"-"`
	SecClientCert          string   `envconfig:"KAFKA_SEC_CLIENT_CERT"`
	SecCACerts             string   `envconfig:"KAFKA_SEC_CA_CERTS"`
	SecSkipVerify          bool     `envconfig:"KAFKA_SEC_SKIP_VERIFY"`
	GenerateDownloadsTopic string   `envconfig:"GENERATE_DOWNLOADS_TOPIC"`
}

var defaultCfg = Config{
	KafkaConfig: KafkaConfig{
		Brokers:                []string{"kafka-1:9092", "kafka-2:9092", "kafka-3:9092"},
		Version:                "1.0.2",
		GenerateDownloadsTopic: "filter-job-submitted",
	},
	Timeout: 30 * time.Second,
	Edition: "time-series",
}

func main() {
	cfg := getConfig()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	log.Info(ctx, "Config", log.Data{"config": cfg})

	pChannels := kafka.CreateProducerChannels()

	pConfig := &kafka.ProducerConfig{
		KafkaVersion:    &cfg.KafkaConfig.Version,
		MaxMessageBytes: &cfg.KafkaConfig.MaxBytes,
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

	// kafka may block, so do work in goroutine (cancel context when done)
	go func() {
		defer cancel()

		prod, err := kafka.NewProducer(ctx, cfg.KafkaConfig.Brokers, cfg.KafkaConfig.GenerateDownloadsTopic, pChannels, pConfig)
		if err != nil {
			log.Error(ctx, "NewProducer failed", err)
			return
		}
		defer func() {
			log.Info(ctx, "closing producer...")
			if err := prod.Close(ctx); err != nil {
				log.Error(ctx, "close failed", err)
			}
			log.Info(ctx, "producer closed")
		}()
		downloadGenerator := &download.Generator{
			Producer:   adapter.NewProducerAdapter(prod),
			Marshaller: schema.GenerateDownloadsEvent,
		}

		log.Info(ctx, "waiting for kafka initialisation...")
		select {
		case <-ctx.Done():
			return
		case <-pChannels.Ready:
		}
		log.Info(ctx, "kafka initialised")

		tmr := time.NewTimer(4 * time.Second) // settle time seems necessary
		select {
		case <-ctx.Done():
			tmr.Stop()
			return
		case <-tmr.C:
		}

		log.Info(ctx, "message send...")
		if err = downloadGenerator.Generate(ctx, cfg.DatasetID, cfg.InstanceID, cfg.Edition, cfg.Version); err != nil {
			log.Error(ctx, "Generate failed", err)
			return
		}

		log.Info(ctx, "message sent, pausing...")
		tmr2 := time.NewTimer(3 * time.Second) // pause before closing, necessary to allow message to depart
		select {
		case <-ctx.Done():
			tmr2.Stop()
			return
		case <-tmr2.C:
		}
	}()

	// wait for kafka work to complete (or timeout, or error)
	select {
	case err := <-pChannels.Errors:
		if err != nil {
			log.Error(ctx, "producer error", err)
		} else {
			log.Info(ctx, "Errors chan closed")
		}
		cancel()
	case <-ctx.Done():
		log.Info(ctx, "context has gone")
		if err := ctx.Err(); err != nil && err != context.Canceled {
			panic(err)
		}
	}
	log.Info(ctx, "2s pause...")
	time.Sleep(2 * time.Second)
	log.Info(ctx, "done")
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
