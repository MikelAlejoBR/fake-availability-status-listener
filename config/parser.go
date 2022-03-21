package config

import (
	"fmt"
	v1 "github.com/redhatinsights/app-common-go/pkg/api/v1"
	"os"
)

// defaultPort is the default port the service will run on.
const defaultPort = "10000"
// sourcesV31Path is the path to the latest API version.
const sourcesV31Path = "api/sources/v3.1"

// KafkaHost is the host of the Kafka instance. Useful for the Kafka reader.
var KafkaHost string
// KafkaPort is the port of the Kafka instance. Useful for the Kafka reader.
var KafkaPort int
// KafkaUrl is the "host:port" URL of the Kafka instance. Useful for the Kafka writer.
var KafkaUrl string
// Port is the port this service will run on.
var Port string
// SourcesApiHealthUrl is the full URL for the "health" endpoint of the sources-api back end.
var SourcesApiHealthUrl string
// SourcesApiUrl is the URL for the sources-api back end, including the "v31Path".
var SourcesApiUrl string

// ParseConfig grabs the URLs for the Kafka and Sources API instances. If Clowder is enabled the Kafka parameters are
// taken from there. Otherwise, it just grabs the variables from the environment.
func ParseConfig() error {
	if v1.IsClowderEnabled() {
		kafkaBroker := v1.LoadedConfig.Kafka.Brokers[0]

		hostname := kafkaBroker.Hostname
		if hostname == "" {
			return fmt.Errorf("configuration missing: Kafka hostname")
		}

		port := kafkaBroker.Port
		if port == nil {
			return fmt.Errorf("configuration missing: Kafka port")
		} else if *port == 0 {
			return fmt.Errorf("configuration missing: Kafka port")
		}

		KafkaUrl = fmt.Sprintf("%s:%d", hostname, port)
	} else {
		hostname := os.Getenv("QUEUE_HOST")
		if hostname == "" {
			return fmt.Errorf("configuration missing: Kafka host")
		}

		port := os.Getenv("QUEUE_PORT")
		if port == "" || port == "0" {
			return fmt.Errorf("configuration missing: Kafka port")
		}

		KafkaUrl = fmt.Sprintf("%s:%s", hostname, port)
	}

	{
		sourcesHost := os.Getenv("SOURCES_API_HOST")
		if sourcesHost == "" {
			return fmt.Errorf("configuration missing: Sources API host")
		}

		sourcesPort := os.Getenv("SOURCES_API_PORT")
		if sourcesPort == "" || sourcesPort == "0" {
			return fmt.Errorf("configuration missing: Sources API port")
		}

		SourcesApiHealthUrl = fmt.Sprintf("%s:%s/health", sourcesHost, sourcesPort)
		SourcesApiUrl = fmt.Sprintf("%s:%s/%s", sourcesHost, sourcesPort, sourcesV31Path)
	}

	port := os.Getenv("PORT")
	if port == "" || port == "0" {
		Port = defaultPort
	} else {
		Port = port
	}

	return nil
}
