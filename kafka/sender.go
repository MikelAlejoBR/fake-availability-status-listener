package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MikelAlejoBR/fake-availability-status-listener/config"
	"github.com/segmentio/kafka-go"
	"log"
	"math/rand"
	"strings"
)

const sourcesStatusTopic = "platform.sources.status"

var kafkaWriter kafka.Writer

type StatusMessage struct {
	ResourceType  string      `json:"resource_type"`
	ResourceID    string      `json:"resource_id"`
	Status        string      `json:"status"`
	Error         string      `json:"error"`
}

func init() {
	kafkaWriter = kafka.Writer{
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

	err = kafkaWriter.WriteMessages(
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
