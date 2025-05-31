FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Install Air for hot reload
RUN go install github.com/air-verse/air@latest
ENV PATH="/go/bin:$PATH"

COPY . .

RUN go build -o main ./cmd/api

EXPOSE 8080

CMD ["air"] 