// Package log implements basic but useful request (access) logging plugin.
package log

import (
	"context"
	"time"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/request"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/miekg/dns"
)

const INSERT_QUERY_SQL = "INSERT INTO queries(ip, qname, qtype, rcode, duration_us, host) VALUES ($1, $2, $3, $4, $5, $6)"

// Logger is a basic request logging plugin.
type Logger struct {
	Next     plugin.Handler
	Hostname string
	Pool     *pgxpool.Pool
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
	go l.Pool.Exec(context.Background(), INSERT_QUERY_SQL, event.IP, event.QName, event.QType, event.Rcode, event.DurationUS, event.NSHostname)

	return status, err
}

// Name implements the Handler interface.
func (l Logger) Name() string { return "timescale" }
