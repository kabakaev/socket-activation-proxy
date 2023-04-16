FROM golang:bullseye as build

WORKDIR /srv

COPY . ./

RUN env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -tags netgo -ldflags '-s -w' -o socket-activation-proxy .


FROM debian:bullseye
COPY --from=build /srv/socket-activation-proxy /bin/socket-activation-proxy
