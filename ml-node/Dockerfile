FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/service .
EXPOSE 9051
CMD ["./service"]
