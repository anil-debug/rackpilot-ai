# syntax=docker/dockerfile:1

FROM golang:1.23.2-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/bmc-simulator ./cmd/bmc-simulator

FROM alpine:3.20.3
ENV RACKPILOT_BMC_FIXTURE_ROOT=/fixtures/dmtf/public-rackmount1
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out/bmc-simulator /usr/local/bin/bmc-simulator
COPY testdata/redfish/dmtf/ /fixtures/dmtf/
EXPOSE 8081
USER nobody
ENTRYPOINT ["/usr/local/bin/bmc-simulator"]
CMD ["-listen", ":8081"]