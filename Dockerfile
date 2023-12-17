FROM golang:1.21-alpine3.19 as base
WORKDIR /lemonai

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY main.go .
COPY internal internal
COPY static static
RUN go build -o main main.go

FROM alpine:3.19
COPY --from=base /lemonai/main main
COPY --from=base /lemonai/static static

ENTRYPOINT [ "/main" ]