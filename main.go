package main

import (
	"encoding/json"
	"fmt"
	"github.com/MikelAlejoBR/fake-availability-status-listener/config"
	"github.com/MikelAlejoBR/fake-availability-status-listener/kafka"
	"github.com/MikelAlejoBR/fake-availability-status-listener/resources"
	"github.com/MikelAlejoBR/fake-availability-status-listener/tenant"
	"io"
	"log"
	"net/http"
)

func main() {
	err := config.ParseConfig()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/health", HandleHealthEndpoint)
	http.HandleFunc("/availability_check", HandleAvailabilityCheck)

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

	w.WriteHeader(http.StatusNoContent)
}

// ResponseSourceId is a struct which grabs the source id that comes in an availability status request.
type ResponseSourceId struct {
	SourceId string `json:"source_id"`
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
	var responseSourceId ResponseSourceId
	err = json.Unmarshal(body, &responseSourceId)
	if err != nil {
		log.Printf(`could not extract source id from availability status request. Request: %v, error: %s`, req, err)
	}

	// Let a routine do the work and get ready for the next request.
	go checkAvailability(responseSourceId.SourceId, xRhIdentity)
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
