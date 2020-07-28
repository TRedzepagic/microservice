package signalization

import (
	"fmt"
	"os"

	"github.com/TRedzepagic/microservice/internal/mail"
	"github.com/TRedzepagic/microservice/internal/ping"
	"github.com/spf13/viper"
)

// Handler does one loop through the hosts when woken up by a signal
func Handler(v *viper.Viper, mailInfoChannel chan<- mail.MailHostInfo, hostChannel <-chan []ping.Host, sigs <-chan os.Signal, configurationPtr *ping.Config, hostToPingChannel chan<- ping.Host) {
	for {
		select {
		case siggy := <-sigs:
			fmt.Printf("You have force activated pinging with %s \n", siggy)
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
		default:
			// Nothing, empty looping
		}
	}
}
