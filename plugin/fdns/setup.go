package fdns

import (
	"context"
	"log"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joeguo/tldextract"
)

func init() {
	caddy.RegisterPlugin("fdns", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	backend := FDNSBackend{}
	c.Next()

	if !c.NextArg() {
		return plugin.Error("fdns", c.ArgErr())
	}
	connString := c.Val()
	log.Println("[fdns] connecting to", connString)

	dbPool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return err
	}
	backend.Pool = dbPool

	extract, _ := tldextract.New("/tmp/tld.cache", false)
	backend.TldExtract = extract

	if c.NextArg() {
		return plugin.Error("fdns", c.ArgErr())
	}

	dnsserver.
		GetConfig(c).
		AddPlugin(func(next plugin.Handler) plugin.Handler {
			backend.Next = next
			return backend
		})

	return nil
}
