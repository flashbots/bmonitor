# stage: build ---------------------------------------------------------

FROM golang:1.22-alpine as build

RUN apk add --no-cache gcc musl-dev linux-headers

WORKDIR /go/src/github.com/flashbots/bmonitor

COPY go.* ./
RUN go mod download

COPY . .

RUN go build -o bin/bmonitor -ldflags "-s -w" github.com/flashbots/bmonitor/cmd

# stage: run -----------------------------------------------------------

FROM alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=build /go/src/github.com/flashbots/bmonitor/bin/bmonitor ./bmonitor

ENTRYPOINT ["/app/bmonitor"]
