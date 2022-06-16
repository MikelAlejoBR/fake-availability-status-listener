package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MikelAlejoBR/fake-availability-status-producer/common"
	"github.com/MikelAlejoBR/fake-availability-status-producer/config"
	"github.com/MikelAlejoBR/fake-availability-status-producer/kafka"
	"github.com/MikelAlejoBR/fake-availability-status-producer/resources"
	"github.com/MikelAlejoBR/fake-availability-status-producer/tenant"
	"io"
	"log"
	"net/http"
)

func main() {
	err := config.ParseConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Kafka once we are sure that the configuration variables have values.
	kafka.Initialize()

	go ListenForAvailabilityStatusRequests()
	http.HandleFunc("/health", HandleHealthEndpoint)
	http.HandleFunc("/availability-check", HandleAvailabilityCheck)

	address := fmt.Sprintf(":%s", config.Port)
	log.Fatal(http.ListenAndServe(address, nil))
}

// HandleHealthEndpoint hits the "/health" endpoint of the sources api back end to see if it is reachable, and sends a
// dummy availability status message to Kafka for the same reason.
func HandleHealthEndpoint(w http.ResponseWriter, _ *http.Request) {
	healthRes, err := http.Get(config.SourcesApiHealthUrl)
	if err != nil {
		log.Printf("[health check path: %s] could not perform health check request: %s", config.SourcesApiHealthUrl, err)

		// Send a response to the caller.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		if _, err = w.Write([]byte(`{"error": "could not perform the health check request to the sources-api back end"}`)); err != nil {
			log.Printf(`error sending the "sources-api back end unreachable" error": %s`, err)
		}

		return
	}

	if healthRes.StatusCode != http.StatusOK {
		log.Printf(`[health check path: %s] unexpected status code received. Expected "%d", got "%d"`, config.SourcesApiHealthUrl, http.StatusOK, healthRes.StatusCode)

		// Send a response to the caller.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		if _, err = w.Write([]byte(`{"error": "sources-api back end returned a non 200 response code."}`)); err != nil {
			log.Printf(`error sending the "sources-api returned a non 200 response code" error": %s`, err)
		}

		return
	}

	if err := kafka.SendStatusMessage("Health", "12345", "invalidXRhIdentity"); err != nil {
		log.Printf(`could not perform health check for the Kafka instance: %s`, err)

		// Send a response to the caller.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		if _, err = w.Write([]byte(`{"error": "could not perform the health check to the Kafka instance"}`)); err != nil {
			log.Printf(`error sending the "Kafka unavailable" error": %s`, err)
		}

		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleAvailabilityCheck checks that the incoming request has the required "x-rh-identity" header, extracts the
// source id from the request, and fires a goroutine which generates random availability status messages for all its
// applications and endpoints, and for the source itself.
func HandleAvailabilityCheck(w http.ResponseWriter, req *http.Request) {
	// Check for the "x-rh-identity" header.
	xRhIdentity := req.Header.Get("x-rh-identity")
	if xRhIdentity == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		_, err := w.Write([]byte(`{"error": "x-rh-identity must have a base64 encoded identity header"}`))
		if err != nil {
			log.Printf(`error sending the "x-rh-identity header required" message: %s`, err)
		}

		return
	}

	w.WriteHeader(http.StatusAccepted)

	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf(`could not read request's body. Body: %s, error: %s`, req.Body, err)
		return
	}

	err = req.Body.Close()
	if err != nil {
		log.Printf(`could not close request's body: %s`, err)
	}

	// Extract the source id of which the availability status must be checked.
	var responseSourceId common.SourceId
	err = json.Unmarshal(body, &responseSourceId)
	if err != nil {
		log.Printf(`could not extract source id from availability status request. Request: %v, error: %s`, req, err)
	}

	// Let a routine do the work and get ready for the next request.
	go checkAvailability(responseSourceId.SourceId, xRhIdentity)
}


// ListenForAvailabilityStatusRequests listens for availability status requests on a specific Kafka topic and checks
// for the availability of all its dependant applications and endpoints.
func ListenForAvailabilityStatusRequests() {
	for {
		message, err := kafka.KafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Printf(`error reading the Kafka message: %s`, err)
			return
		}

		var xRhIdentity string
		for _, header := range message.Headers {
			if header.Key == "x-rh-identity" {
				xRhIdentity = string(header.Value)
			}
		}

		if xRhIdentity == "" {
			log.Printf(`ignoring unauthenticated Kafka mesasge: %v`, message)
			return
		}

		var sourceId common.SourceId
		err = json.Unmarshal(message.Value, &sourceId)
		if err != nil {
			log.Printf(`error unmarshalling source id from Kafka message: %s`, err)
			return
		}

		checkAvailability(sourceId.SourceId, xRhIdentity)
	}
}

// checkAvailability grabs a source's applications and endpoints and sends random availability status messages to Kafka
// for each resource. It also sends an availability status message for the source itself.
func checkAvailability(sourceId string, xRhIdentity string) {
	// Check for the existence of the source.
	sourceExists, err := resources.SourceExists(sourceId, xRhIdentity)
	if err != nil {
		log.Println(err)
		return
	}

	if !sourceExists {
		log.Printf(`[account_number: %s][source_id: %s] source does not exist`, tenant.DecodeAccountNumber(xRhIdentity), sourceId)
		return
	}

	// Give the source a random status.
	if err = kafka.SendStatusMessage("Source", sourceId, xRhIdentity); err != nil {
		log.Println(err)
	}

	// Give the source's apps a random status.
	appIds, err := resources.GetApplicationIds(sourceId, xRhIdentity)
	if err != nil {
		log.Println(err)
		return
	}

	for _, app := range appIds.Data {
		if err := kafka.SendStatusMessage("Application", app.Id, xRhIdentity); err != nil {
			log.Println(err)
		}
	}

	// Give the source's endpoints a random status.
	endpointIds, err := resources.GetResourceIds(sourceId, "endpoints", xRhIdentity)
	if err != nil {
		log.Println(err)
		return
	}

	for _, endpoint := range endpointIds.Data {
		if err = kafka.SendStatusMessage("Endpoint", endpoint.Id, xRhIdentity); err != nil {
			log.Println(err)
		}
	}
}
