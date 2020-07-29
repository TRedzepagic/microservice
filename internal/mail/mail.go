package mail

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/irnes/go-mailer"
	"gopkg.in/yaml.v2"
)

// Config data contract
type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

// HostInfo data contract
type HostInfo struct {
	HostIterationAddress      string
	HostIterationPingInterval string
	Recipients                []string
}

// GetMail gets personal configuration from another file
func GetMail(path string) Config {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("error opening configuration", err.Error())
	}

	var mailconfiguration Config
	err = yaml.Unmarshal(data, &mailconfiguration)

	if err != nil {
		fmt.Println("error unmarshalling ", err.Error())
	}
	return mailconfiguration
}

// Sender to downed hosts, starts in a thread and waits for downed hosts
func Sender(mailconf *Config, mailHostInfoChannel <-chan HostInfo) {
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
		// Blocks until downed host arrives
		hostStructFromChan := <-mailHostInfoChannel
		mail := mailer.NewMail()
		mail.FromName = "Go Mailer - Redzep Microservice"
		mail.From = config.User
		for _, recipientIteration := range hostStructFromChan.Recipients {
			mail.SetTo(recipientIteration)
		}
		mail.Subject = "Admin notice : Server Down"
		mail.Body = "Your server is down. Host Address: " + hostStructFromChan.HostIterationAddress + " " + "Host pinging interval:" + hostStructFromChan.HostIterationPingInterval

		fmt.Println("Not actually mailing, testing to avoid clutter : ")
		fmt.Println("Detected e-mails : ")
		fmt.Println(hostStructFromChan.Recipients)

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
