FROM golang:1.15.7-alpine AS builder

RUN mkdir /build
ADD go.mod go.sum cmd/words-api/ /build/
WORKDIR /build
RUN go build

FROM alpine
RUN adduser -S -D -H -h /app appuser
USER appuser
COPY --from=builder /build/words-api /app/
COPY assets/ /app/assets
WORKDIR /app
CMD ["./words-api"]