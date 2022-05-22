package fdns

import (
	"context"
	"log"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/miekg/dns"
)

const Name = "fdns"
const GET_RECORDS_SQL = "SELECT content,ttl FROM records WHERE name = $1 AND type = $2"

type FDNSBackend struct {
	Pool  *pgxpool.Pool
	Debug bool
	Next  plugin.Handler
}

func (b FDNSBackend) Name() string { return Name }

func (b FDNSBackend) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	_ = state

	a := new(dns.Msg)
	a.SetReply(r)
	a.Compress = true
	a.Authoritative = true

	log.Println("[fdns]", state.QName(), dns.TypeToString[state.QType()])

	rows, err := b.Pool.Query(context.Background(), GET_RECORDS_SQL, state.QName(), dns.TypeToString[state.QType()])
	if err != nil {
		log.Print("[fdns]", err)
		return dns.RcodeServerFailure, err
	}
	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			return dns.RcodeServerFailure, err
		}

		hdr := dns.RR_Header{Name: state.QName(), Rrtype: state.QType(), Class: state.QClass(), Ttl: uint32(row[1].(int32))}
		if !strings.HasSuffix(hdr.Name, ".") {
			hdr.Name += "."
		}

		var rr dns.RR
		switch state.QType() {
		case dns.TypeA:
			rr = &dns.A{Hdr: hdr, A: net.ParseIP(row[0].(string))}
		}

		a.Answer = append(a.Answer, rr)
	}

	if len(a.Answer) == 0 {
		return plugin.NextOrFailure(b.Name(), b.Next, ctx, w, r)
	}

	return dns.RcodeSuccess, w.WriteMsg(a)
}
