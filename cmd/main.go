package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/hantdev/hermina"
	"github.com/hantdev/hermina/examples/simple"
	"github.com/hantdev/hermina/pkg/http"
	"github.com/hantdev/hermina/pkg/mqtt"
	"github.com/hantdev/hermina/pkg/mqtt/websocket"
	"github.com/hantdev/hermina/pkg/session"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

const (
	mqttWithoutTLS = "HERMINA_MQTT_WITHOUT_TLS_"
	mqttWithTLS    = "HERMINA_MQTT_WITH_TLS_"
	mqttWithmTLS   = "HERMINA_MQTT_WITH_MTLS_"

	mqttWSWithoutTLS = "HERMINA_MQTT_WS_WITHOUT_TLS_"
	mqttWSWithTLS    = "HERMINA_MQTT_WS_WITH_TLS_"
	mqttWSWithmTLS   = "HERMINA_MQTT_WS_WITH_MTLS_"

	httpWithoutTLS = "HERMINA_HTTP_WITHOUT_TLS_"
	httpWithTLS    = "HERMINA_HTTP_WITH_TLS_"
	httpWithmTLS   = "HERMINA_HTTP_WITH_MTLS_"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(logHandler)

	handler := simple.New(logger)

	var interceptor session.Interceptor

	// Loading .env file to environment
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	// hermina server Configuration for MQTT without TLS
	mqttConfig, err := hermina.NewConfig(env.Options{Prefix: mqttWithoutTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for MQTT without TLS
	mqttProxy := mqtt.New(mqttConfig, handler, interceptor, logger)
	g.Go(func() error {
		return mqttProxy.Listen(ctx)
	})

	// hermina server Configuration for MQTT with TLS
	mqttTLSConfig, err := hermina.NewConfig(env.Options{Prefix: mqttWithTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for MQTT with TLS
	mqttTLSProxy := mqtt.New(mqttTLSConfig, handler, interceptor, logger)
	g.Go(func() error {
		return mqttTLSProxy.Listen(ctx)
	})

	//  hermina server Configuration for MQTT with mTLS
	mqttMTLSConfig, err := hermina.NewConfig(env.Options{Prefix: mqttWithmTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for MQTT with mTLS
	mqttMTlsProxy := mqtt.New(mqttMTLSConfig, handler, interceptor, logger)
	g.Go(func() error {
		return mqttMTlsProxy.Listen(ctx)
	})

	// hermina server Configuration for MQTT over Websocket without TLS
	wsConfig, err := hermina.NewConfig(env.Options{Prefix: mqttWSWithoutTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for MQTT over Websocket without TLS
	wsProxy := websocket.New(wsConfig, handler, interceptor, logger)
	g.Go(func() error {
		return wsProxy.Listen(ctx)
	})

	// hermina server Configuration for MQTT over Websocket with TLS
	wsTLSConfig, err := hermina.NewConfig(env.Options{Prefix: mqttWSWithTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for MQTT over Websocket with TLS
	wsTLSProxy := websocket.New(wsTLSConfig, handler, interceptor, logger)
	g.Go(func() error {
		return wsTLSProxy.Listen(ctx)
	})

	// hermina server Configuration for MQTT over Websocket with mTLS
	wsMTLSConfig, err := hermina.NewConfig(env.Options{Prefix: mqttWSWithmTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for MQTT over Websocket with mTLS
	wsMTLSProxy := websocket.New(wsMTLSConfig, handler, interceptor, logger)
	g.Go(func() error {
		return wsMTLSProxy.Listen(ctx)
	})

	// hermina server Configuration for HTTP without TLS
	httpConfig, err := hermina.NewConfig(env.Options{Prefix: httpWithoutTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for HTTP without TLS
	httpProxy, err := http.NewProxy(httpConfig, handler, logger)
	if err != nil {
		panic(err)
	}
	g.Go(func() error {
		return httpProxy.Listen(ctx)
	})

	// hermina server Configuration for HTTP with TLS
	httpTLSConfig, err := hermina.NewConfig(env.Options{Prefix: httpWithTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for HTTP with TLS
	httpTLSProxy, err := http.NewProxy(httpTLSConfig, handler, logger)
	if err != nil {
		panic(err)
	}
	g.Go(func() error {
		return httpTLSProxy.Listen(ctx)
	})

	// hermina server Configuration for HTTP with mTLS
	httpMTLSConfig, err := hermina.NewConfig(env.Options{Prefix: httpWithmTLS})
	if err != nil {
		panic(err)
	}

	// hermina server for HTTP with mTLS
	httpMTLSProxy, err := http.NewProxy(httpMTLSConfig, handler, logger)
	if err != nil {
		panic(err)
	}
	g.Go(func() error {
		return httpMTLSProxy.Listen(ctx)
	})

	g.Go(func() error {
		return StopSignalHandler(ctx, cancel, logger)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("hermina service terminated with error: %s", err))
	} else {
		logger.Info("hermina service stopped")
	}
}

func StopSignalHandler(ctx context.Context, cancel context.CancelFunc, logger *slog.Logger) error {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGINT, syscall.SIGABRT)
	select {
	case <-c:
		cancel()
		return nil
	case <-ctx.Done():
		return nil
	}
}
