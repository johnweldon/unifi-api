package unifi

import (
	"encoding/json"
	"fmt"
	"net/http"

	caddy "github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/johnweldon/unifi-scheduler/pkg/nats"
	"github.com/johnweldon/unifi-scheduler/pkg/unifi"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

func (a *API) ServeHTTP(rw http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	a.logger.Debug("ServeHTTP",
		zap.String("method", r.Method),
		zap.String("path", r.RequestURI),
		zap.String("remote addr", r.RemoteAddr),
		zap.String("user-agent", r.Header.Get("user-agent")),
	)

	a.router.ServeHTTP(rw, r)

	if len(rw.Header().Get("not-found")) == 0 {
		return nil
	}

	return next.ServeHTTP(rw, r)
}

func (a *API) registerRouter(ctx caddy.Context) {
	router := httprouter.New()
	router.GET("/", a.index)
	router.GET("/list", a.list)
	router.PUT("/block/:name", a.block)
	router.PUT("/unblock/:name", a.unblock)

	router.NotFound = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("not-found", "true")
	})

	a.router = router
}

func (a *API) index(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	rw.Header().Set("content-type", "application/json")
	fmt.Fprintf(rw, "{}")
}

func (a *API) list(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	opts := []nats.ClientOpt{nats.OptNATSUrl(a.NATSURL)}
	s := nats.NewSubscriber(opts...)

	var into []unifi.Client
	if err := s.Get(nats.DetailBucket(baseSubject), nats.ActiveKey, &into); err != nil {
		a.logger.Error("list: unable to get clients", zap.Error(err))

		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	rw.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(rw).Encode(into); err != nil {
		a.logger.Error("list: unable to json encode clients", zap.Error(err))

		rw.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func (a *API) block(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ses, err := a.initSession()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	names, err := ses.GetNames()
	if err != nil {
		a.logger.Error("block: unable to get names", zap.Error(err))

		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	victim := params.ByName("name")
	var blocked, notBlocked []unifi.MAC

	if macs, ok := names[victim]; ok {
		for _, mac := range macs {
			if _, err = ses.Block(mac); err != nil {
				a.logger.Warn("block: unable to block mac",
					zap.Error(err),
					zap.String("mac", string(mac)),
				)
				notBlocked = append(notBlocked, mac)
			} else {
				blocked = append(blocked, mac)
			}
		}
	}

	rw.Header().Set("content-type", "application/json")
	if err = json.NewEncoder(rw).Encode(map[string]interface{}{"blocked": blocked, "not_blocked": notBlocked}); err != nil {
		a.logger.Error("block: unable to json encode results", zap.Error(err))

		rw.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func (a *API) unblock(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ses, err := a.initSession()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	names, err := ses.GetNames()
	if err != nil {
		a.logger.Error("unblock: unable to get names", zap.Error(err))

		rw.WriteHeader(http.StatusInternalServerError)

		return
	}

	victim := params.ByName("name")
	var unblocked, notUnblocked []unifi.MAC

	if macs, ok := names[victim]; ok {
		for _, mac := range macs {
			if _, err = ses.Unblock(mac); err != nil {
				a.logger.Warn("unblock: unable to unblock mac",
					zap.Error(err),
					zap.String("mac", string(mac)),
				)
				notUnblocked = append(notUnblocked, mac)
			} else {
				unblocked = append(unblocked, mac)
			}
		}
	}

	rw.Header().Set("content-type", "application/json")
	if err = json.NewEncoder(rw).Encode(map[string]interface{}{"unblocked": unblocked, "not_unblocked": notUnblocked}); err != nil {
		a.logger.Error("unblock: unable to json encode results", zap.Error(err))

		rw.WriteHeader(http.StatusInternalServerError)

		return
	}
}

func (a *API) initSession() (*unifi.Session, error) {
	ses := &unifi.Session{
		Endpoint: a.BaseURL,
		Username: a.Username,
		Password: a.Password,
	}

	if err := ses.Initialize(); err != nil {
		a.logger.Error("initSession: unable to Initialize session",
			zap.Error(err),
			zap.String("endpoint", a.BaseURL),
			zap.String("username", a.Username),
		)

		return nil, err
	}

	if msg, err := ses.Login(); err != nil {
		a.logger.Error("initSession: unable to login session",
			zap.Error(err),
			zap.String("message", msg),
			zap.String("endpoint", a.BaseURL),
			zap.String("username", a.Username),
		)

		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	return ses, nil
}

const baseSubject = "unifi"

var _ caddyhttp.MiddlewareHandler = (*API)(nil)
