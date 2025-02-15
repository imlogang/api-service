FROM golang:1.23 AS builder
WORKDIR /go-api-service
COPY go.mod ./
RUN go mod tidy
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o /go-api-service/app .

# Start a new stage to build the final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /go-api-service
COPY --from=builder /go-api-service/app .
EXPOSE 8080
CMD ["./app"]
