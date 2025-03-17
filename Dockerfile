FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go-api-service ./cmd/main.go

# Start a new stage to build the final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go-api-service /go-api-service
COPY /cmd/website /cmd/website
EXPOSE 8080
ENTRYPOINT ["/go-api-service"]