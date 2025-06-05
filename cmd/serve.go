package main

import (
	"slices"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/flashbots/bmonitor/config"
	"github.com/flashbots/bmonitor/server"
)

const (
	categoryMonitor = "monitor"
	categoryServer  = "server"
)

func CommandServe(cfg *config.Config) *cli.Command {
	monitorBuilders := &cli.StringSlice{}
	monitorPeers := &cli.StringSlice{}

	monitorFlags := []cli.Flag{
		&cli.StringSliceFlag{
			Category:    strings.ToUpper(categoryMonitor),
			Destination: monitorBuilders,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryMonitor) + "_BUILDERS"},
			Name:        categoryMonitor + "-builders",
			Usage:       "list of monitored builder rpc endpoints in the format `name=url`",
		},

		&cli.DurationFlag{
			Category:    strings.ToUpper(categoryMonitor),
			Destination: &cfg.Monitor.Interval,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryMonitor) + "_INTERVAL"},
			Name:        categoryMonitor + "-interval",
			Usage:       "`interval` at which to query builders for their status",
			Value:       5 * time.Second,
		},

		&cli.StringSliceFlag{
			Category:    strings.ToUpper(categoryMonitor),
			Destination: monitorPeers,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryMonitor) + "_PEERS"},
			Name:        categoryMonitor + "-peers",
			Usage:       "list of monitored builder rpc endpoints in the format `label=ip`",
		},

		&cli.DurationFlag{
			Category:    strings.ToUpper(categoryMonitor),
			Destination: &cfg.Monitor.Timeout,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryMonitor) + "_TIMEOUT"},
			Name:        categoryMonitor + "-timeout",
			Usage:       "timeout `duration` for rpc queries",
			Value:       500 * time.Millisecond,
		},
	}

	serverFlags := []cli.Flag{
		&cli.StringFlag{
			Category:    strings.ToUpper(categoryServer),
			Destination: &cfg.Server.ListenAddress,
			EnvVars:     []string{envPrefix + strings.ToUpper(categoryServer) + "_LISTEN_ADDRESS"},
			Name:        categoryServer + "-listen-address",
			Usage:       "`host:port` for the server to listen on",
			Value:       "0.0.0.0:8080",
		},
	}

	flags := slices.Concat(
		monitorFlags,
		serverFlags,
	)

	return &cli.Command{
		Name:  "serve",
		Usage: "run bmonitor server",
		Flags: flags,

		Before: func(_ *cli.Context) error {
			cfg.Monitor.Builders = monitorBuilders.Value()
			cfg.Monitor.Peers = monitorPeers.Value()
			return cfg.Validate()
		},

		Action: func(_ *cli.Context) error {
			s, err := server.New(cfg)
			if err != nil {
				return err
			}
			return s.Run()
		},
	}
}
