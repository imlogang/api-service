FROM cimg/go:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api-service ./cmd/main.go

FROM scratch
COPY --from=builder /app/api-service /api-service
ENTRYPOINT ["/api-service"]