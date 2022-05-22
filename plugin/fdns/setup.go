package fdns

import (
	"context"
	"log"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/jackc/pgx/v4/pgxpool"
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

	for c.NextBlock() {
		x := c.Val()
		switch x {
		case "debug":
			args := c.RemainingArgs()
			for _, v := range args {
				switch v {
				case "db":
					// backend.DB = backend.DB.Debug()
				}
			}
			backend.Debug = true
			log.Println(Name, "enable log", args)
		default:
			return plugin.Error("fdns", c.Errf("unexpected '%v' command", x))
		}
	}

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
