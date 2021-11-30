package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kafka "github.com/ONSdigital/dp-kafka/v2"
	"github.com/ONSdigital/dp-kafka/v2/avro"
	"github.com/ONSdigital/log.go/v2/log"
)

// AuditEvent represents the structure of the audit message received
type AuditEvent struct {
	CreatedAt    int64  `avro:"created_at"`
	RequestID    string `avro:"request_id"`
	Identity     string `avro:"identity"`
	CollectionID string `avro:"collection_id"`
	Path         string `avro:"path"`
	Method       string `avro:"method"`
	StatusCode   int32  `avro:"status_code"`
	QueryParam   string `avro:"query_param"`
}

// CreatedAtTime returns a time.Time representation of the CreatedAt field of an AuditEvent struct
func (a *AuditEvent) CreatedAtTime() time.Time {
	var sec, nanosec int64
	sec = a.CreatedAt / 1e3
	nanosec = (a.CreatedAt % 1e3) * 1e6
	return time.Unix(sec, nanosec).UTC()
}

var audit = `{
	"type": "record",
	"name": "audit",
	"fields": [
	  {"name": "created_at", "type": "long", "logicalType": "timestamp-millis"},
	  {"name": "request_id", "type": "string", "default": ""},
	  {"name": "identity", "type": "string", "default": ""},
	  {"name": "collection_id", "type": "string", "default": ""},
	  {"name": "path", "type": "string", "default": ""},
	  {"name": "method", "type": "string", "default": ""},
	  {"name": "status_code", "type": "int", "default": 0},
	  {"name": "query_param", "type": "string", "default": ""}
	]
  }`

// AuditSchema is the Avro schema for Audit messages.
var AuditSchema = &avro.Schema{
	Definition: audit,
}

// Action represents stats of the consumed actions
type Action struct {
	Attempted    int
	Successful   int
	Unsuccessful int
	Total        int
}

const (
	topic         = "audit"
	consumerGroup = "check-audit"
)

const (
	attempted    = "attempted"
	successful   = "successful"
	unsuccessful = "unsuccessful"
)

var kafkaBrokers string
var kafkaVersion = "1.0.2"

func main() {

	// Get context and parse input
	ctx := context.Background()
	flag.StringVar(&kafkaBrokers, "kafka-brokers", kafkaBrokers, "kafka broker addresses, comma separated")
	flag.Parse()

	// Validate
	if kafkaBrokers == "" {
		err := errors.New("missing kafka brokers, must be comma separated")
		log.Error(ctx, "", err, log.Data{"kafka_brokers": kafkaBrokers})
		return
	}
	kafkaBrokerList := strings.Split(kafkaBrokers, ",")

	// Create Consumer with channels
	cgChannels := kafka.CreateConsumerGroupChannels(1)
	cgConfig := &kafka.ConsumerGroupConfig{KafkaVersion: &kafkaVersion}
	consumer, err := kafka.NewConsumerGroup(
		ctx, kafkaBrokerList, topic, consumerGroup, cgChannels, cgConfig)
	if err != nil {
		log.Fatal(ctx, "[KAFKA-TEST] Fatal error creating consumer.", err)
		os.Exit(1)
	}

	// OS Signals channel
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	paths := make(map[string]Action)
	for {
		select {
		case message := <-consumer.Channels().Upstream:
			event, err := readMessage(message.GetData())
			if err != nil {
				log.Error(ctx, "", err, log.Data{"schema": "failed to unmarshal event"})
				break
			}

			createdAtTime := event.CreatedAtTime()
			log.Info(ctx, "received message", log.Data{"audit_event": event, "created_at_time": createdAtTime})

			addResult(paths, event)

			message.Commit()
		case <-signals:
			log.Info(ctx, "audit stats", log.Data{"audit": paths})
			os.Exit(0)
		}
	}
}

func readMessage(eventValue []byte) (*AuditEvent, error) {
	var e AuditEvent

	if err := AuditSchema.Unmarshal(eventValue, &e); err != nil {
		return nil, err
	}

	return &e, nil
}

func addResult(paths map[string]Action, event *AuditEvent) {
	action, ok := paths[event.Path]
	if !ok {
		paths[event.Path] = Action{
			Attempted:    0,
			Successful:   0,
			Unsuccessful: 0,
			Total:        0,
		}
	}

	switch event.StatusCode {
	case 0:
		action.Attempted++
	case http.StatusInternalServerError:
		action.Unsuccessful++
	default:
		action.Successful++
	}

	action.Total++
	paths[event.Path] = action
}
