FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod tidy
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o app .

# Start a new stage to build the final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/app .
EXPOSE 8080
CMD ["./app"]
