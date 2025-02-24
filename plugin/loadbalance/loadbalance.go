// Package loadbalance shuffles A, AAAA and MX records.
package loadbalance

import (
	"github.com/miekg/dns"
)

// RoundRobinResponseWriter is a response writer that shuffles A, AAAA and MX records.
type RoundRobinResponseWriter struct{ dns.ResponseWriter }

// WriteMsg implements the dns.ResponseWriter interface.
func (r *RoundRobinResponseWriter) WriteMsg(res *dns.Msg) error {
	if res.Rcode != dns.RcodeSuccess {
		return r.ResponseWriter.WriteMsg(res)
	}

	if res.Question[0].Qtype == dns.TypeAXFR || res.Question[0].Qtype == dns.TypeIXFR {
		return r.ResponseWriter.WriteMsg(res)
	}

	res.Answer = roundRobin(res.Answer)
	res.Ns = roundRobin(res.Ns)
	res.Extra = roundRobin(res.Extra)

	return r.ResponseWriter.WriteMsg(res)
}

func roundRobin(in []dns.RR) []dns.RR {
	cname := []dns.RR{}
	address := []dns.RR{}
	mx := []dns.RR{}
	ns := []dns.RR{}
	rest := []dns.RR{}
	for _, r := range in {
		switch r.Header().Rrtype {
		case dns.TypeCNAME:
			cname = append(cname, r)
		case dns.TypeA, dns.TypeAAAA:
			address = append(address, r)
		case dns.TypeMX:
			mx = append(mx, r)
		case dns.TypeNS:
			ns = append(ns, r)
		default:
			rest = append(rest, r)
		}
	}

	roundRobinShuffle(address)
	roundRobinShuffle(mx)
	roundRobinShuffle(ns)

	out := append(cname, rest...)
	out = append(out, ns...)
	out = append(out, address...)
	out = append(out, mx...)
	return out
}

func roundRobinShuffle(records []dns.RR) {
	switch l := len(records); l {
	case 0, 1:
		break
	case 2:
		if dns.Id()%2 == 0 {
			records[0], records[1] = records[1], records[0]
		}
	default:
		for j := 0; j < l; j++ {
			p := j + (int(dns.Id()) % (l - j))
			if j == p {
				continue
			}
			records[j], records[p] = records[p], records[j]
		}
	}
}

// Write implements the dns.ResponseWriter interface.
func (r *RoundRobinResponseWriter) Write(buf []byte) (int, error) {
	// Should we pack and unpack here to fiddle with the packet... Not likely.
	log.Warning("RoundRobin called with Write: not shuffling records")
	n, err := r.ResponseWriter.Write(buf)
	return n, err
}
