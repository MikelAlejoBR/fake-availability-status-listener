# fake-availability-status-listener
Pretends to be a fake availability status listener which plays nice with what the sources-api back end expects. It runs
on the `10000` port by default.

## Usage

Once built with `go build`, be sure to provide the following environment variables:

* `PORT` — If set, it runs on the specified port. Example value: `10000`
* `QUEUE_HOST` — If Clowder is enabled it's not needed. Example value: `localhost`
* `QUEUE_PORT` — If Clowder is enabled it's not needed. Example value: `9092`
* `SOURCES_API_HOST` Example value: `http://localhost`
* `SOURCES_API_PORT` Example value: `SOURCES_API_PORT=8000`

QUEUE_HOST=localhost;QUEUE_PORT=9092;SOURCES_API_HOST=http://localhost;SOURCES_API_PORT=8000