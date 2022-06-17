Pretends to be a fake availability status listener which plays nice with what the sources-api back end expects. It runs
on the `10000` port by default.

## Usage

Once built with `go build`, be sure to provide the following environment variables when running the binary:

* `PORT` — If set, it runs on the specified port. Default value: `8000`
* `QUEUE_HOST` — If Clowder is enabled it's not needed. Example value: `localhost`
* `QUEUE_PORT` — If Clowder is enabled it's not needed. Example value: `9092`
* `SOURCES_API_HOST` Example value: `http://localhost`
* `SOURCES_API_PORT` Example value: `SOURCES_API_PORT=10000`

It has two endpoints:

* `/health` which checks if the "sources-api" back end is reachable.
* `/availability-status` which expects the `{"source_id": "<id>"}` payload to start the process.

It also listens on the `platform.topological-inventory.operations-satellite` Kafka topic, but again, it only cares
for a payload with a "source_id" key. The rest of the keys are ignored.

Once an availability status update request has been received, the binary randomly picks one of the four valid
availability statuses: "available", "in_progress", "partially_available" and "unavailable". If it chooses one of the
latter two, it also includes an error message.

Finally, the status update message gets posted to the `platform.sources.status` Kafka topic, which is the one the
sources-api back end is listening on.
