FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go-api-service ./cmd/main.go

# Start a new stage to build the final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /
COPY --from=builder /go-api-service /go-api-service
EXPOSE 8080
ENTRYPOINT ["/go-api-service"]