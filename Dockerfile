FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o orchids-api ./cmd/server

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/orchids-api /app/
COPY --from=builder /app/data /app/data/
COPY --from=builder /app/web /app/web/

EXPOSE 3002

ENV PORT=3002
ENV DEBUG_ENABLED=false

CMD ["./orchids-api"]
