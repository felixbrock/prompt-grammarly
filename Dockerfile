# go
FROM golang:1.21-alpine3.16 as go_base
WORKDIR /lemonai

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY main.go .
RUN go build -o main main.go

FROM alpine:3.16
COPY --from=go_base /lemonai/main /main
ENTRYPOINT [ "/main" ]

# bun
FROM oven/bun:latest as bun_base

FROM bun_base AS install
RUN mkdir -p /tmp/prod
COPY package.json bun.lockb /tmp/prod/
RUN cd /tmp/prod && bun install --frozen-lockfile --production

# then copy all (non-ignored) project files into the image
FROM bun_base AS prerelease
COPY --from=install /tmp/prod/node_modules node_modules
COPY . .

NODE_ENV=prod
RUN bun test
RUN bun run build

FROM bun_base AS release
COPY --from=install /tmp/prod/node_modules node_modules
COPY --from=prerelease /package.json .
COPY --from=prerelease /tailwind.config.js .

# run
USER bun
EXPOSE 8000/tcp
ENTRYPOINT [ "bun", "start" ]
