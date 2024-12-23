#base go image

FROM golang:1.22-alpine as builder

RUN mkdir /app

COPY . /app

WORKDIR /app

RUN CGO_ENABLED=0 go build -o k2gs ./cmd/api

RUN chmod +x ./k2gs

#build a tiny docker image
FROM alpine:latest

RUN mkdir /app

COPY --from=builder /app /app

WORKDIR /app

CMD ["/app/k2gs"]