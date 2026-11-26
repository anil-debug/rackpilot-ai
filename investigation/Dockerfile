# syntax=docker/dockerfile:1

FROM golang:1.23.2-alpine AS build
RUN apk add --no-cache ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/investigator ./cmd/investigator

FROM alpine:3.20.3
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out/investigator /usr/local/bin/investigator
EXPOSE 8084
USER nobody
ENTRYPOINT ["/usr/local/bin/investigator"]
CMD ["-listen", ":8084"]