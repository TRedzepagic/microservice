package configuration

import (
	"fmt"
	"log"
	"os"

	"github.com/TRedzepagic/microservice/internal/mail"
	"github.com/TRedzepagic/microservice/internal/ping"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// getConfig gets configuration file of hosts
func getConfig(v *viper.Viper, conf *ping.Config) ping.Config {
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

// Initialize processes general file initialization
func Initialize() (mail.Config, ping.Config, *viper.Viper) {

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
	var conf ping.Config
	configUnmarshalError := v.Unmarshal(&conf)
	if configUnmarshalError != nil {
		log.Fatalf("unable to decode into struct, %v", configUnmarshalError)
	}
	mailConfiguration := mail.GetMail(path)
	return mailConfiguration, conf, v
}

// ConfigWatcher watches for configuration file changes
func ConfigWatcher(v *viper.Viper, hostChannel chan []ping.Host, configurationPtr *ping.Config) {
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
