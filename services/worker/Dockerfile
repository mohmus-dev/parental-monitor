FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o worker .

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /src/worker /app/worker
CMD ["/app/worker"]
