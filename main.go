package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
	"time"
)

func main() {
	listenAddress := flag.String("l", "localhost:8080", "listen address")
	backendAddress := flag.String("b", "localhost:80", "backend address")
	backendCommand := flag.String("c", "/entrypoint", "backend command")
	backendTimeoutSeconds := flag.Duration("timeout", 10*time.Second, "timeout for no new connections before stopping the command, defaults to 30s")

	flag.Parse()

	listener, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		panic(err)
	}

	connections_wg := sync.WaitGroup{}
	var backendIsRunning sync.Mutex
	for {
		frontendConn, err := listener.Accept()
		if err != nil {
			log.Println("error accepting connection", err)
			continue
		}
		log.Println("new connection from ", frontendConn.RemoteAddr())
		connections_wg.Add(1)
		startBackend(&connections_wg, backendCommand, &backendIsRunning)
		go func() {
			defer frontendConn.Close()
			defer connections_wg.Done()
			var backendConn net.Conn
			for { // Connect to the backend service with a retry.
				backendConn, err = net.DialTimeout("tcp", *backendAddress, 900*time.Millisecond)
				if err == nil {
					defer backendConn.Close()
					break
				}
				log.Println("error while connecting to remote address:", err)
				time.Sleep(time.Second)
			}
			closer := make(chan struct{}, 2)
			go copy(closer, backendConn, frontendConn)
			go copy(closer, frontendConn, backendConn)
			<-closer
			log.Println("connection complete, waiting for a timeout", frontendConn.RemoteAddr())
			time.Sleep(*backendTimeoutSeconds)
			log.Println("connection complete, timeout has expired", frontendConn.RemoteAddr())
		}()
	}
}

func copy(closer chan struct{}, dst io.Writer, src io.Reader) {
	_, _ = io.Copy(dst, src)
	closer <- struct{}{} // connection is closed, send signal to stop proxy
}

func startBackend(wg *sync.WaitGroup, backendCommand *string, backendIsRunning *sync.Mutex) error {
	if !backendIsRunning.TryLock() {
		return nil
	}

	go func(wg *sync.WaitGroup, backendIsRunning *sync.Mutex) {
		// See https://github.com/golang/go/issues/27505
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		ctx, cancel := context.WithCancel(context.Background())
		cmd := exec.CommandContext(ctx, *backendCommand)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
			Pdeathsig: syscall.SIGTERM,
		}
		// Retry to start the backend process.
		for {
			err := cmd.Start()
			if err == nil {
				break
			}
			log.Println("cannot start the backend: ", err.Error())
		}
		// This goroutine will wait for all the connections to finish and then will kill the backend process.
		go func(wg *sync.WaitGroup, cancel context.CancelFunc) {
			wg.Wait()
			syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
			cancel()
		}(wg, cancel)
		cmd.Wait()
		log.Println("the backend was stopped")
		time.Sleep(time.Second) // to prevent too many backend command executions
		backendIsRunning.Unlock()
	}(wg, backendIsRunning)

	return nil
}
