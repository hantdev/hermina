package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/hantdev/hermina"
	"github.com/hantdev/hermina/pkg/session"
	mptls "github.com/hantdev/hermina/pkg/tls"
	"golang.org/x/sync/errgroup"
)

const contentType = "application/json"

// ErrMissingAuthentication returned when no basic or Authorization header is set.
var ErrMissingAuthentication = errors.New("missing authorization")

func (p Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Metrics and health endpoints are served directly.
	if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
		p.target.ServeHTTP(w, r)
		return
	}

	if !strings.HasPrefix(r.URL.Path, p.config.PathPrefix) {
		http.NotFound(w, r)
		return
	}

	username, password, ok := r.BasicAuth()
	switch {
	case ok:
		break
	case r.Header.Get("Authorization") != "":
		password = r.Header.Get("Authorization")
	default:
		encodeError(w, http.StatusBadGateway, ErrMissingAuthentication)
		return
	}

	s := &session.Session{
		Password: []byte(password),
		Username: username,
	}
	ctx := session.NewContext(r.Context(), s)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		encodeError(w, http.StatusBadRequest, err)
		p.logger.Error("Failed to read body", slog.Any("error", err))
		return
	}
	if err := r.Body.Close(); err != nil {
		encodeError(w, http.StatusInternalServerError, err)
		p.logger.Error("Failed to close body", slog.Any("error", err))
		return
	}

	// r.Body is reset to ensure it can be safely copied by httputil.ReverseProxy.
	// no close method is required since NopClose Close() always returns nill.
	r.Body = io.NopCloser(bytes.NewBuffer(payload))
	if err := p.session.AuthConnect(ctx); err != nil {
		encodeError(w, http.StatusUnauthorized, err)
		p.logger.Error("Failed to authorize connect", slog.Any("error", err))
		return
	}
	if err := p.session.Publish(ctx, &r.RequestURI, &payload); err != nil {
		encodeError(w, http.StatusBadRequest, err)
		p.logger.Error("Failed to publish", slog.Any("error", err))
		return
	}
	p.target.ServeHTTP(w, r)
}

func encodeError(w http.ResponseWriter, defStatusCode int, err error) {
	hpe, ok := err.(HTTPProxyError)
	if !ok {
		hpe = NewHTTPProxyError(defStatusCode, err)
	}
	w.WriteHeader(hpe.StatusCode())
	w.Header().Set("Content-Type", contentType)
	if err := json.NewEncoder(w).Encode(err); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Proxy represents HTTP Proxy.
type Proxy struct {
	config  hermina.Config
	target  *httputil.ReverseProxy
	session session.Handler
	logger  *slog.Logger
}

func NewProxy(config hermina.Config, handler session.Handler, logger *slog.Logger) (Proxy, error) {
	target, err := url.Parse(config.Target)
	if err != nil {
		return Proxy{}, err
	}

	return Proxy{
		config:  config,
		target:  httputil.NewSingleHostReverseProxy(target),
		session: handler,
		logger:  logger,
	}, nil
}

func (p Proxy) Listen(ctx context.Context) error {
	l, err := net.Listen("tcp", p.config.Address)
	if err != nil {
		return err
	}

	if p.config.TLSConfig != nil {
		l = tls.NewListener(l, p.config.TLSConfig)
	}
	status := mptls.SecurityStatus(p.config.TLSConfig)

	p.logger.Info(fmt.Sprintf("HTTP proxy server started at %s%s with %s", p.config.Address, p.config.PathPrefix, status))

	var server http.Server
	g, ctx := errgroup.WithContext(ctx)

	mux := http.NewServeMux()
	mux.Handle(p.config.PathPrefix, p)
	server.Handler = mux

	g.Go(func() error {
		return server.Serve(l)
	})

	g.Go(func() error {
		<-ctx.Done()
		return server.Close()
	})
	if err := g.Wait(); err != nil {
		p.logger.Info(fmt.Sprintf("HTTP proxy server at %s%s with %s exiting with errors", p.config.Address, p.config.PathPrefix, status), slog.String("error", err.Error()))
	} else {
		p.logger.Info(fmt.Sprintf("HTTP proxy server at %s%s with %s exiting...", p.config.Address, p.config.PathPrefix, status))
	}
	return nil
}
