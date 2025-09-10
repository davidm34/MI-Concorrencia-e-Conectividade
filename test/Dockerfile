FROM golang:1.25-alpine

WORKDIR /app

COPY go.mod .
COPY stress.go .

RUN go build -o test stress.go

CMD ["./test"]
