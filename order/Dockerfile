FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o order-service ./cmd

EXPOSE 8080
EXPOSE 50052

CMD ["./order-service"]