package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"strings"
	"syscall"
	"time"

	"github.com/flashbots/bmonitor/config"
	"github.com/flashbots/bmonitor/httplogger"
	"github.com/flashbots/bmonitor/logutils"
	"github.com/flashbots/bmonitor/metrics"
	"github.com/flashbots/bmonitor/utils"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	otelapi "go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

type Server struct {
	cfg *config.Config

	failure chan error

	logger *zap.Logger
	server *http.Server

	builders map[string]*ethclient.Client
	peers    map[string]string
	ticker   *time.Ticker
}

func New(cfg *config.Config) (*Server, error) {
	builders := make(map[string]*ethclient.Client, len(cfg.Monitor.Builders))
	for _, b := range cfg.Monitor.Builders {
		parts := strings.Split(b, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid builder: %s", b)
		}
		name := strings.TrimSpace(parts[0])
		rpc, err := ethclient.Dial(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		builders[name] = rpc
	}

	peers := make(map[string]string, 0)
	for _, peer := range cfg.Monitor.Peers {
		parts := strings.Split(peer, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid peer config: %s", peer)
		}
		label := strings.TrimSpace(parts[0])
		if len(label) == 0 {
			return nil, fmt.Errorf("invalid peer label: %s", peer)
		}
		ip := net.ParseIP(strings.TrimSpace(parts[1]))
		if ip == nil {
			if len(label) == 0 {
				return nil, fmt.Errorf("invalid peer ip: %s", peer)
			}
		}
		if _, known := peers[ip.String()]; known {
			if len(label) == 0 {
				return nil, fmt.Errorf("duplicate ip: %s vs %s",
					peer, fmt.Sprintf("%s=%s", label, peers[ip.String()]),
				)
			}
		}
		peers[ip.String()] = label
	}

	s := &Server{
		builders: builders,
		cfg:      cfg,
		failure:  make(chan error, 1),
		logger:   zap.L(),
		peers:    peers,
		ticker:   time.NewTicker(cfg.Monitor.Interval),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHealthcheck)
	mux.Handle("/metrics", promhttp.Handler())
	handler := httplogger.Middleware(s.logger, mux)

	s.server = &http.Server{
		Addr:              cfg.Server.ListenAddress,
		ErrorLog:          logutils.NewHttpServerErrorLogger(s.logger),
		Handler:           handler,
		MaxHeaderBytes:    1024,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	return s, nil
}

func (s *Server) Run() error {
	l := s.logger
	ctx := logutils.ContextWithLogger(context.Background(), l)

	if err := metrics.Setup(ctx); err != nil {
		return err
	}

	s.initializePeersMetrics(ctx)

	go func() { // run the server
		l.Info("Builder monitor server is going up...",
			zap.String("server_listen_address", s.cfg.Server.ListenAddress),
		)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.failure <- err
		}
		l.Info("Builder monitor server is down")
	}()

	go func() { // run the monitor loop
		for {
			s.monitor(ctx, <-s.ticker.C)
		}
	}()

	errs := []error{}
	{ // wait until termination or internal failure
		terminator := make(chan os.Signal, 1)
		signal.Notify(terminator, os.Interrupt, syscall.SIGTERM)

		select {
		case stop := <-terminator:
			l.Info("Stop signal received; shutting down...",
				zap.String("signal", stop.String()),
			)
		case err := <-s.failure:
			l.Error("Internal failure; shutting down...",
				zap.Error(err),
			)
			errs = append(errs, err)
		exhaustErrors:
			for { // exhaust the errors
				select {
				case err := <-s.failure:
					l.Error("Extra internal failure",
						zap.Error(err),
					)
					errs = append(errs, err)
				default:
					break exhaustErrors
				}
			}
		}
	}

	{ // stop the monitor loop
		s.ticker.Stop()
	}

	{ // stop the server
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			l.Error("Builder monitor server shutdown failed",
				zap.Error(err),
			)
		}
	}

	{ // close the clients
		for _, rpc := range s.builders {
			rpc.Close()
		}
	}

	return utils.FlattenErrors(errs)
}

func (s *Server) initializePeersMetrics(ctx context.Context) {
	for builder := range s.builders {
		for _, typ := range []string{"loopback", "internal", "external"} {
			metrics.PeersCount.Record(ctx, 0, otelapi.WithAttributes(
				attribute.KeyValue{Key: "builder", Value: attribute.StringValue(builder)},
				attribute.KeyValue{Key: "type", Value: attribute.StringValue(typ)},
			))
		}
		for _, label := range s.peers {
			metrics.PeersCount.Record(ctx, 0, otelapi.WithAttributes(
				attribute.KeyValue{Key: "builder", Value: attribute.StringValue(builder)},
				attribute.KeyValue{Key: "type", Value: attribute.StringValue("labelled")},
				attribute.KeyValue{Key: "label", Value: attribute.StringValue(label)},
			))
		}
	}
}
