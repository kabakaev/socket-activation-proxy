
.PHONY: build container push

build:
	go build -tags netgo -o "$(CURDIR)/socket-activation-proxy" "$(CURDIR)"

container:
	podman build -t docker.io/kabakaev/socket-activation-proxy "$(CURDIR)"

push: REVISION := $(shell git rev-parse --short=8 HEAD)
push: DATE := $(shell date -I)
push: container
	podman tag docker.io/kabakaev/socket-activation-proxy docker.io/kabakaev/socket-activation-proxy:$(DATE)-$(REVISION)
	podman push docker.io/kabakaev/socket-activation-proxy:$(DATE)-$(REVISION)
