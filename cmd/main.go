package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/TRedzepagic/microservice/internal/configuration"
	"github.com/TRedzepagic/microservice/internal/mail"
	"github.com/TRedzepagic/microservice/internal/ping"
	"github.com/TRedzepagic/microservice/internal/signalization"
)

var wg sync.WaitGroup

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	runtime.GOMAXPROCS(runtime.NumCPU()) // 8 Core limit

	// Get process PID
	fmt.Printf("PID: %d \n", os.Getpid())

	// General initialization, we get two configurations.
	mailconf, conf, v := configuration.Initialize()

	// Initialization of channels
	// This channel is used for relaying info to the mailing thread.
	mailInfoChannel := make(chan mail.HostInfo)

	hostChannel := make(chan []ping.Host)
	hostToPingChannel := make(chan ping.Host)

	// Watch for user defined signal ((10) - SIGUSR1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1)

	// Goroutines/concurrent workers..
	go ping.ContinuousLooper(v, mailInfoChannel, hostChannel, &conf, hostToPingChannel)
	// ThreadWorker blocks until Looper sends host over hostToPingChannel
	go ping.ThreadWorker(mailInfoChannel, hostToPingChannel)
	// Sender blocks until it receives host over mailInfoChannel
	go mail.Sender(&mailconf, mailInfoChannel)
	// Handler blocks until it receives a signal over sigs channel
	go signalization.Handler(v, mailInfoChannel, hostChannel, sigs, &conf, hostToPingChannel)
	go configuration.ConfigWatcher(v, hostChannel, &conf)

	<-stop
}
