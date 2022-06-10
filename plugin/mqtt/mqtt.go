// Package log implements basic but useful request (access) logging plugin.
package log

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/request"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/miekg/dns"
)

// Logger is a basic request logging plugin.
type Logger struct {
	Next     plugin.Handler
	Mqtt     mqtt.Client
	Hostname string
}

type Event struct {
	IP         string `json:"ip"`
	QName      string `json:"qname"`
	QType      string `json:"qtype"`
	Rcode      string `json:"rcode"`
	DurationUS int64  `json:"duration_us"`
	NSHostname string `json:"ns_hostname"`
}

// ServeDNS implements the plugin.Handler interface.
func (l Logger) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	rw := dnstest.NewRecorder(w)
	status, err := plugin.NextOrFailure(l.Name(), l.Next, ctx, rw, r)

	event := Event{
		IP:         state.IP(),
		QName:      state.QName(),
		QType:      state.Type(),
		Rcode:      dns.RcodeToString[rw.Msg.Rcode],
		DurationUS: time.Since(rw.Start).Microseconds(),
		NSHostname: l.Hostname,
	}
	json, _ := json.Marshal(event)
	jstring := string(json)
	log.Println(jstring)

	token := l.Mqtt.Publish("fdns", 0, false, jstring)
	token.Wait()

	return status, err
}

// Name implements the Handler interface.
func (l Logger) Name() string { return "log" }
