# TCP proxy with socket activation functionality

This service implements a socket activation proxy.

A first connection triggers startup of a backend server.

If all connections are closed, then the backend service gets stopped.

This proxy is meant be used as an entrypoint of a container,
because [vanila Kubernetes cannot stop unused containers](https://github.com/kubernetes/kubernetes/issues/484) yet.

For socket activation proxy on Linux without Kubernetes,
see [systemd-socket-proxyd](https://www.freedesktop.org/software/systemd/man/systemd-socket-proxyd.html).
