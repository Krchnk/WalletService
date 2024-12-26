FROM golang:1.23.4
WORKDIR /app
COPY . .
WORKDIR /app/cmd/wallet-service
RUN go mod tidy && go build -o /app/main
CMD ["/app/main"]
