FROM registry.access.redhat.com/ubi8/ubi:8.5-214 as build
WORKDIR /build

RUN dnf -y --disableplugin=subscription-manager install go

COPY go.mod .
RUN go mod download

COPY . .
RUN go build -o fake-availability-status-listener .

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5-218
COPY --from=build /build/fake-availability-status-listener /fake-availability-status-listener
ENTRYPOINT ["/fake-availability-status-listener"]
