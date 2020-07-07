package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/irnes/go-mailer"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var wg sync.WaitGroup

type host struct {
	Address      string   `yaml:"address"`
	Pinginterval string   `yaml:"pinginterval"`
	Recipients   []string `yaml:"recipients"`
}

type config struct {
	Hosts []host `yaml:"hosts"`
}

type mailConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type mailHostInfo struct {
	hostIterationAddress      string
	hostIterationPingInterval string
	recipients                []string
}

// getConfig gets configuration file of hosts
func getConfig(v *viper.Viper, conf *config) config {
	viperReadErr := v.ReadInConfig() // Find and read the config file
	if viperReadErr != nil {
		// Handle errors reading the config file
		panic(fmt.Errorf("fatal error in config file :  %s ", viperReadErr))
	}
	configUnmarshalError := v.Unmarshal(&conf)
	if configUnmarshalError != nil {
		log.Fatalf("unable to decode into struct, %v", configUnmarshalError)

	}
	return *conf

}

// getMail gets personal configuration from another file
func getMail(path string) mailConfig {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error opening configuration", err.Error())
	}

	var mailconfiguration mailConfig
	err = yaml.Unmarshal(data, &mailconfiguration)

	if err != nil {
		fmt.Println("error unmarshalling ", err.Error())
	}
	return mailconfiguration
}

// pingThreadWorker pings hosts
func pingThreadWorker(mailHostInfoChannel chan<- mailHostInfo, hostToPingChannel <-chan host) {
	for {
		hostIteration := <-hostToPingChannel
		// Ping syscall, -c is ping count, -i is interval, -w is timeout
		out, _ := exec.Command("ping", hostIteration.Address, "-c 5", hostIteration.Pinginterval, "-w 2").Output()
		fmt.Println("PING INTERVAL OF : " + hostIteration.Address + " SET TO : " + hostIteration.Pinginterval)
		if (strings.Contains(string(out), "Destination Host Unreachable")) || (strings.Contains(string(out), "100% packet loss")) {
			fmt.Println("HOST " + hostIteration.Address + " IS DOWN, SENDING MAIL..")

			// instantiate a structure representing host info, shorthand
			mailHostInfoStruct := mailHostInfo{
				recipients:                hostIteration.Recipients,
				hostIterationAddress:      hostIteration.Address,
				hostIterationPingInterval: hostIteration.Pinginterval}
			mailHostInfoChannel <- mailHostInfoStruct
		} else {
			fmt.Println("Host ping successful! Ignoring mailing protocol.")
			fmt.Println(string(out))
		}
	}
}

// Mailing to downed hosts, starts in a thread and waits for downed hosts
func mailThread(mailconf *mailConfig, mailHostInfoChannel <-chan mailHostInfo) {
	config := mailer.Config{
		Host: mailconf.Host,
		Port: mailconf.Port,
		User: mailconf.User,
		Pass: mailconf.Pass,
	}

	// Checking purposes, will probably use more adequate protection going forward..
	fmt.Println("Configuration of user mail: ")
	fmt.Println("Host: " + mailconf.Host)
	fmt.Println("Mailing port: " + strconv.Itoa(mailconf.Port))
	fmt.Println("User: Hidden")
	fmt.Println("Pass: Hidden")

	for {

		hostStructFromChan := <-mailHostInfoChannel
		mail := mailer.NewMail()
		mail.FromName = "Go Mailer - Redzep Microservice"
		mail.From = config.User
		for _, recipientIteration := range hostStructFromChan.recipients {
			mail.SetTo(recipientIteration)
		}
		mail.Subject = "Admin notice : Server Down"
		mail.Body = "Your server is down. Host Address: " + hostStructFromChan.hostIterationAddress + " " + "Host pinging interval:" + hostStructFromChan.hostIterationPingInterval

		fmt.Println("Not actually mailing, testing to avoid clutter : ")
		fmt.Println("Detected e-mails : ")
		fmt.Println(hostStructFromChan.recipients)

		// used for actual mailing, uncomment when needed

		// mailerino := mailer.NewMailer(config, true)
		// err := mailerino.Send(mail)
		// if err != nil {
		// 	println(err)
		// } else {
		// 	fmt.Println("Mail sent to : ")
		// 	fmt.Println(hostStructFromChan.recipients)
		// }
	}

}

// initialize processes general file initialization
func initialize() (mailConfig, config, *viper.Viper) {

	// personal mail info path
	path := os.Getenv("MAILCONF")
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	viperReadErr := v.ReadInConfig() // Find and read the config file
	if viperReadErr != nil {
		// Handle errors reading the config file
		panic(fmt.Errorf("fatal error in config file :  %s ", viperReadErr))
	}
	var conf config
	configUnmarshalError := v.Unmarshal(&conf)
	if configUnmarshalError != nil {
		log.Fatalf("unable to decode into struct, %v", configUnmarshalError)
	}
	mailConfiguration := getMail(path)
	return mailConfiguration, conf, v
}

// configWatcher watches for configuration file changes
func configWatcher(v *viper.Viper, hostChannel chan []host, configurationPtr *config) {
	wg.Add(1)
	// Trying to avoid data races with viper but to no avail, viper seems inherently broken
	// Experimenting with locking proved useless. (github.com/spf13/viper/issues/174)
	// "- race" will report a data race when changing the configuration file.
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("NEW EVENT !!! : " + e.Op.String())
		log.Println("config file changed", e.Name)
		log.Println("reloading..", e.Name)
		*configurationPtr = getConfig(v, configurationPtr)
		hostChannel <- configurationPtr.Hosts
	})
}

// continuousLooper loops continuously, sends tasks (hosts) to ping workers
func continuousLooper(v *viper.Viper, mailInfoChannel chan<- mailHostInfo, hostChannel <-chan []host, configurationPtr *config, hostToPingChannel chan<- host) {
	clk := 30 * time.Second
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

// signalHandler does one loop through the hosts when woken up by a signal
func signalHandler(v *viper.Viper, mailInfoChannel chan<- mailHostInfo, hostChannel <-chan []host, sigs <-chan os.Signal, configurationPtr *config, hostToPingChannel chan<- host) {
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

func main() {
	wg.Add(1)

	runtime.GOMAXPROCS(runtime.NumCPU()) // 8 Core limit

	// Get process PID
	fmt.Printf("PID: %d \n", os.Getpid())

	// General initialization, we get two configurations.
	mailconf, conf, v := initialize()

	// Initialization of channels
	// This channel is used for relaying info to the mailing thread.
	mailInfoChannel := make(chan mailHostInfo)

	hostChannel := make(chan []host)
	hostToPingChannel := make(chan host)

	// Watch for user defined signal ((10) - SIGUSR1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1)

	// Goroutines
	go mailThread(&mailconf, mailInfoChannel)
	go continuousLooper(v, mailInfoChannel, hostChannel, &conf, hostToPingChannel)
	go signalHandler(v, mailInfoChannel, hostChannel, sigs, &conf, hostToPingChannel)
	go configWatcher(v, hostChannel, &conf)
	go pingThreadWorker(mailInfoChannel, hostToPingChannel)
	wg.Wait()
}
