FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o payment-service cmd/main.go

EXPOSE 8081
EXPOSE 50051

CMD ["./payment-service"]