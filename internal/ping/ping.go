package ping

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/TRedzepagic/microservice/internal/mail"
	"github.com/spf13/viper"
)

// Host data type
type Host struct {
	Address      string   `yaml:"address"`
	Pinginterval string   `yaml:"pinginterval"`
	Recipients   []string `yaml:"recipients"`
}

// Config wrapper for hosts
type Config struct {
	Hosts []Host `yaml:"hosts"`
}

// ThreadWorker pings hosts
func ThreadWorker(mailHostInfoChannel chan<- mail.HostInfo, hostToPingChannel <-chan Host) {
	for {
		hostIteration := <-hostToPingChannel
		// Ping syscall, -c is ping count, -i is interval, -w is timeout
		out, _ := exec.Command("ping", hostIteration.Address, "-c 5", hostIteration.Pinginterval, "-w 2").Output()
		fmt.Println("PING INTERVAL OF : " + hostIteration.Address + " SET TO : " + hostIteration.Pinginterval)
		if (strings.Contains(string(out), "Destination Host Unreachable")) || (strings.Contains(string(out), "100% packet loss")) {
			fmt.Println("HOST " + hostIteration.Address + " IS DOWN, SENDING MAIL..")

			// instantiate a structure representing host info, shorthand
			mailHostInfoStruct := mail.HostInfo{
				Recipients:                hostIteration.Recipients,
				HostIterationAddress:      hostIteration.Address,
				HostIterationPingInterval: hostIteration.Pinginterval}
			mailHostInfoChannel <- mailHostInfoStruct
		} else {
			fmt.Println("Host ping successful! Ignoring mailing protocol.")
			fmt.Println(string(out))
		}
	}
}

// ContinuousLooper loops continuously, sends tasks (hosts) to ping workers
func ContinuousLooper(v *viper.Viper, mailInfoChannel chan<- mail.HostInfo, hostChannel <-chan []Host, configurationPtr *Config, hostToPingChannel chan<- Host) {
	clk := 5 * time.Second
	ticker := time.NewTicker(clk)
	fmt.Printf("Pinging of hosts will occur every %f seconds. \n", clk.Seconds())

	for range ticker.C {
		select {
		// Config change occurred, reloaded hosts
		case hostChannelSlice := <-hostChannel:
			for _, hostIteration := range hostChannelSlice {
				hostToPingChannel <- hostIteration
			}
		// Default behavior, no config change, check host perpetually
		default:
			for _, hostIteration := range configurationPtr.Hosts {
				hostToPingChannel <- hostIteration

			}
		}
	}
}
