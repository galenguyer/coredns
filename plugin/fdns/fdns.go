package fdns

import (
	"context"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joeguo/tldextract"
	"github.com/miekg/dns"
)

const Name = "fdns"
const GET_RECORDS_SQL = "SELECT content,type,ttl FROM records WHERE name = $1 AND (type = $2 OR type = 'CNAME')"
const GET_SOA_SQL = "SELECT content,type,ttl FROM records WHERE name = $1 AND type = 'SOA'"
const GET_ZONE_SQL = "SELECT count(id) FROM zones WHERE id = $1"

type FDNSBackend struct {
	Pool       *pgxpool.Pool
	TldExtract *tldextract.TLDExtract
	Debug      bool
	Next       plugin.Handler
}

func (b FDNSBackend) Name() string { return Name }

func (b FDNSBackend) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	// Extract the zone and ensure we're authoritative for it
	extracted := b.TldExtract.Extract(strings.ToLower(strings.TrimRight(state.QName(), ".")))
	zone := extracted.Root + "." + extracted.Tld + "."
	var count int
	b.Pool.QueryRow(context.Background(), GET_ZONE_SQL, zone).Scan(&count)
	if count == 0 {
		log.Println("[fdns] not authoritative for", zone)
		return dns.RcodeRefused, nil
	}

	a := new(dns.Msg)
	a.SetReply(r)
	a.Compress = true
	a.Authoritative = true

	qName := state.QName()
	if !strings.HasSuffix(qName, ".") {
		qName += "."
	}

	rows, err := b.Pool.Query(context.Background(), GET_RECORDS_SQL, strings.ToLower(qName), dns.TypeToString[state.QType()])
	if err != nil {
		log.Print("[fdns]", err)
		return dns.RcodeServerFailure, err
	}
	defer rows.Close()

	for rows.Next() {
		row, err := rows.Values()
		if err != nil {
			log.Print("[fdns]", err)
			continue
		}
		aType := dns.StringToType[row[1].(string)]

		hdr := dns.RR_Header{Name: qName, Rrtype: aType, Class: state.QClass(), Ttl: uint32(row[2].(int32))}

		log.Println("[fdns]", hdr.Name, dns.TypeToString[aType], row[0].(string))
		var rr dns.RR
		switch aType {
		case dns.TypeA:
			rr = &dns.A{Hdr: hdr, A: net.ParseIP(row[0].(string))}
		case dns.TypeAAAA:
			rr = &dns.AAAA{Hdr: hdr, AAAA: net.ParseIP(row[0].(string))}
		case dns.TypeCNAME:
			rr = &dns.CNAME{Hdr: hdr, Target: row[0].(string)}
		case dns.TypeNS:
			rr = &dns.NS{Hdr: hdr, Ns: row[0].(string)}
		case dns.TypeTXT:
			rr = &dns.TXT{Hdr: hdr, Txt: strings.Split(row[0].(string), " ")}
		case dns.TypeSOA:
			rr = &dns.SOA{Hdr: hdr}
			if !parseSOA(rr.(*dns.SOA), row[0].(string)) {
				rr = nil
			}
		}

		if rr != nil {
			a.Answer = append(a.Answer, rr)
		}
	}
	if rows.Err() != nil {
		log.Print("[fdns]", rows.Err())
		return dns.RcodeServerFailure, rows.Err()
	}

	if len(a.Answer) == 0 {
		code, err := plugin.NextOrFailure(b.Name(), b.Next, ctx, w, r)
		if err != nil && err.Error() == "plugin/fdns: no next plugin found" {
			rows, err = b.Pool.Query(context.Background(), GET_SOA_SQL, zone)
			if err != nil {
				log.Print("[fdns]", err)
				return dns.RcodeServerFailure, err
			}
			defer rows.Close()
			for rows.Next() {
				row, err := rows.Values()
				if err != nil {
					log.Print("[fdns]", err)
					return dns.RcodeServerFailure, err
				}

				hdr := dns.RR_Header{Name: qName, Rrtype: dns.TypeSOA, Class: state.QClass(), Ttl: uint32(row[2].(int32))}

				var rr dns.RR
				rr = &dns.SOA{Hdr: hdr}
				if !parseSOA(rr.(*dns.SOA), row[0].(string)) {
					rr = nil
				}

				if rr != nil {
					a.Ns = append(a.Ns, rr)
				} else {
					log.Print("[fdns]", "failed to parse SOA")
					return dns.RcodeServerFailure, nil
				}
			}
		} else {
			return code, err
		}
	}

	return dns.RcodeSuccess, w.WriteMsg(a)
}

func parseSOA(rr *dns.SOA, line string) bool {
	splites := strings.Split(line, " ")
	if len(splites) < 7 {
		return false
	}
	rr.Ns = splites[0]
	rr.Mbox = splites[1]
	if i, err := strconv.Atoi(splites[2]); err != nil {
		return false
	} else {
		rr.Serial = uint32(i)
	}
	if i, err := strconv.Atoi(splites[3]); err != nil {
		return false
	} else {
		rr.Refresh = uint32(i)
	}
	if i, err := strconv.Atoi(splites[4]); err != nil {
		return false
	} else {
		rr.Retry = uint32(i)
	}
	if i, err := strconv.Atoi(splites[5]); err != nil {
		return false
	} else {
		rr.Expire = uint32(i)
	}
	if i, err := strconv.Atoi(splites[6]); err != nil {
		return false
	} else {
		rr.Minttl = uint32(i)
	}
	return true
}
