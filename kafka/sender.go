package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MikelAlejoBR/fake-availability-status-producer/config"
	"github.com/segmentio/kafka-go"
	"log"
	"math/rand"
	"strings"
)

// satelliteStatusTopic is the Kafka topic from which we will read Satellite endpoint status availability requests.
const satelliteStatusTopic = "platform.topological-inventory.operations-satellite"
// sourcesStatusTopic is the Kafka topic to which we will write all the status messages.
const sourcesStatusTopic = "platform.sources.status"

// KafkaReader is the reader we will use to receive status availability requests.
var KafkaReader *kafka.Reader

// KafkaWriter is the writer we will use to produce status availability updates.
var KafkaWriter kafka.Writer

// StatusMessage is the structure that the sources-api expects to process a status availability update.
type StatusMessage struct {
	ResourceType  string      `json:"resource_type"`
	ResourceID    string      `json:"resource_id"`
	Status        string      `json:"status"`
	Error         string      `json:"error"`
}

// Initialize the Kafka reader and writer.
func Initialize() {
	KafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:                []string{config.KafkaUrl},
		Topic: satelliteStatusTopic,
	})

	KafkaWriter = kafka.Writer{
		Addr: kafka.TCP(config.KafkaUrl),
		Topic: sourcesStatusTopic,
		Balancer: &kafka.LeastBytes{},
	}
}

// availabilityStatuses holds the availability statuses that a resource might be in.
var availabilityStatuses = [4]string{
	"available",
	"in_progress",
	"partially_available",
	"unavailable",
}

// errorMessages is an array with random error messages for the availability statuses.
var errorMessages = [8]string{
	"network error",
	"could not reach the resource",
	"internal resource error",
	"database not connected",
	"insufficient permissions",
	"not authorized",
	"missing required headers for authentication",
	"invalid uri provided",
}


// generateStatusMessage generates a random status message for the given resourceType and resourceId. If the generated
// availability status is "partially_available" or "unavailable", a random error message is picked as well to send it
// along.
func generateStatusMessage(resourceType string, resourceId string) *StatusMessage {
	idx := rand.Intn(3)

	var errorMsg string
	if idx > 1 {
		errIdx := rand.Intn(7)
		errorMsg = errorMessages[errIdx]
	}

	return &StatusMessage{
		ResourceType: resourceType,
		ResourceID: resourceId,
		Status: availabilityStatuses[idx],
		Error: errorMsg,
	}
}

// SendStatusMessage generates a random status message for the given resourceType and resourceId and sends it over to
// the Kafka instance.
func SendStatusMessage(resourceType string, resourceId string, xRhIdentity string) error {
	statusMessage := generateStatusMessage(resourceType, resourceId)

	message, err := json.Marshal(statusMessage)
	if err != nil {
		return fmt.Errorf(`could not marshal status message for %s "%s": %s`, strings.ToLower(resourceType), resourceId, err)
	}

	// Set the "event_type" header to let the sources api know that it should process this Kafka message.
	eventTypeHeader := kafka.Header{
		Key:   "event_type",
		Value: []byte("availability_status"),
	}

	xRhIdentityHeader := kafka.Header{
		Key: "x-rh-identity",
		Value: []byte(xRhIdentity),
	}

	err = KafkaWriter.WriteMessages(
		context.Background(),
		kafka.Message{
			Headers: []kafka.Header{eventTypeHeader, xRhIdentityHeader},
			Value: message,
		},
	)
	if err != nil {
		return fmt.Errorf(`[resource_type: %s][resource_id: %s]: could not send Kafka message: %s`, resourceType, resourceId, err)
	}

	log.Printf(`kafka message sent for %s "%s" with status "%s"`, strings.ToLower(resourceType), resourceId, statusMessage.Status)
	return nil
}
