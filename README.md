# microservice
Microservice practice project

## How to run

### Cloning
You can "git clone" my repo with (Entire repository):

```
"git clone https://github.com/TRedzepagic/microservice.git"
```

This program checks for host availability via ICMP pings. If the host is down, the program will send an email to the administrator of the host. We can override the pinging interval with a user signal (SIGUSR). 

Host configuration is kept in a .yaml file (/configs). Sender configuration is kept in a different directory (WiP on making it safer). 

Config file can be reloaded on-the-fly. No processes mentioned is blocking and every process is concurrent.


