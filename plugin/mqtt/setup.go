package log

import (
	"log"
	"os"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func init() {
	caddy.RegisterPlugin("mqtt", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("[mqtt] Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("[mqtt] Connect lost: %v", err)
}

func setup(c *caddy.Controller) error {
	logger := Logger{}
	hostname, _ := os.Hostname()
	logger.Hostname = hostname
	c.Next()

	if !c.NextArg() {
		return plugin.Error("mqtt", c.ArgErr())
	}

	connString := c.Val()
	log.Println("[mqtt] connecting to", connString)
	opts := mqtt.NewClientOptions()
	opts.AddBroker(connString)
	opts.SetClientID(hostname)
	// opts.SetUsername("emqx")
	// opts.SetPassword("public")
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println(token.Error())
	}
	logger.Mqtt = client

	if c.NextArg() {
		return plugin.Error("mqtt", c.ArgErr())
	}

	dnsserver.
		GetConfig(c).
		AddPlugin(func(next plugin.Handler) plugin.Handler {
			logger.Next = next
			return logger
		})

	return nil
}

// func setup(c *caddy.Controller) error {
// 	rules, err := logParse(c)
// 	if err != nil {
// 		return plugin.Error("log", err)
// 	}

// 	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
// 		return Logger{Next: next, Rules: rules, repl: replacer.New()}
// 	})

// 	return nil
// }

// func logParse(c *caddy.Controller) ([]Rule, error) {
// 	var rules []Rule

// 	for c.Next() {
// 		args := c.RemainingArgs()
// 		length := len(rules)

// 		switch len(args) {
// 		case 0:
// 			// Nothing specified; use defaults
// 			rules = append(rules, Rule{
// 				NameScope: ".",
// 				Format:    DefaultLogFormat,
// 				Class:     make(map[response.Class]struct{}),
// 			})
// 		case 1:
// 			rules = append(rules, Rule{
// 				NameScope: dns.Fqdn(args[0]),
// 				Format:    DefaultLogFormat,
// 				Class:     make(map[response.Class]struct{}),
// 			})
// 		default:
// 			// Name scopes, and maybe a format specified
// 			format := DefaultLogFormat

// 			if strings.Contains(args[len(args)-1], "{") {
// 				format = args[len(args)-1]
// 				format = strings.Replace(format, "{common}", CommonLogFormat, -1)
// 				format = strings.Replace(format, "{combined}", CombinedLogFormat, -1)
// 				args = args[:len(args)-1]
// 			}

// 			for _, str := range args {
// 				rules = append(rules, Rule{
// 					NameScope: dns.Fqdn(str),
// 					Format:    format,
// 					Class:     make(map[response.Class]struct{}),
// 				})
// 			}
// 		}

// 		// Class refinements in an extra block.
// 		classes := make(map[response.Class]struct{})
// 		for c.NextBlock() {
// 			switch c.Val() {
// 			// class followed by combinations of all, denial, error and success.
// 			case "class":
// 				classesArgs := c.RemainingArgs()
// 				if len(classesArgs) == 0 {
// 					return nil, c.ArgErr()
// 				}
// 				for _, c := range classesArgs {
// 					cls, err := response.ClassFromString(c)
// 					if err != nil {
// 						return nil, err
// 					}
// 					classes[cls] = struct{}{}
// 				}
// 			default:
// 				return nil, c.ArgErr()
// 			}
// 		}
// 		if len(classes) == 0 {
// 			classes[response.All] = struct{}{}
// 		}

// 		for i := len(rules) - 1; i >= length; i-- {
// 			rules[i].Class = classes
// 		}
// 	}

// 	return rules, nil
// }
