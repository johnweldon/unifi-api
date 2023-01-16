package unifi

import (
	"errors"
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(&API{})
	httpcaddyfile.RegisterHandlerDirective("unifi_api", parseCaddyfile)
}

type API struct {
	BaseURL  string `json:"base_url,omitempty"`
	NATSURL  string `json:"nats_url,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	logger *zap.Logger
	router *httprouter.Router
}

func (API) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.unifi_api",
		New: func() caddy.Module { return new(API) },
	}
}

func (a *API) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "base_url":
				if !d.AllArgs(&a.BaseURL) {
					return d.ArgErr()
				}

			case "nats_url":
				if !d.AllArgs(&a.NATSURL) {
					return d.ArgErr()
				}

			case "username":
				if !d.AllArgs(&a.Username) {
					return d.ArgErr()
				}

			case "password":
				if !d.AllArgs(&a.Password) {
					return d.ArgErr()
				}

			default:
				return fmt.Errorf("unexepected token %q", d.Val())
			}
		}
	}

	return nil
}

func (a *API) Provision(ctx caddy.Context) error {
	a.logger = ctx.Logger()

	a.registerRouter(ctx)

	return nil
}

func (a *API) Validate() error {
	if a.logger == nil {
		return errors.New("logger not initialized")
	}

	return nil
}

// parseCaddyfile unmarshals tokens from h into a new Middleware.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	a := &API{}
	err := a.UnmarshalCaddyfile(h.Dispenser)
	return a, err
}

var (
	_ caddy.Provisioner     = (*API)(nil)
	_ caddy.Validator       = (*API)(nil)
	_ caddyfile.Unmarshaler = (*API)(nil)
)
