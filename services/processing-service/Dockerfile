FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod ./
COPY . .
RUN go mod tidy
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o processing-service .

FROM alpine:latest

RUN apk --no-cache add ffmpeg ca-certificates

WORKDIR /root/

COPY --from=builder /app/processing-service .

RUN mkdir -p /root/temp

CMD ["./processing-service"]
