package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/flashbots/bmonitor/utils"
)

type Monitor struct {
	Builders []string      `yaml:"builders"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

var (
	errMonitorInvalidBuilder  = errors.New("invalid builder")
	errMonitorInvalidInterval = errors.New("invalid monitoring interval (must be non-zero and up to 1h)")
	errMonitorInvalidTimeout  = errors.New("invalid monitoring timeout (must be non-zero, up to 1m, and less than monitoring interval)")
)

func (cfg *Monitor) Validate() error {
	errs := make([]error, 0)

	{ // builders
		for _, builder := range cfg.Builders {
			parts := strings.Split(strings.TrimSpace(builder), "=")
			if len(parts) != 2 {
				errs = append(errs, fmt.Errorf("%w: invalid format (must be `name=url`): %s",
					errMonitorInvalidBuilder, builder,
				))
			}
			if _, err := url.Parse(strings.TrimSpace(parts[1])); err != nil {
				errs = append(errs, fmt.Errorf("%w: %s: invalid url: %w",
					errMonitorInvalidBuilder, builder, err,
				))
			}
		}
	}

	{ // interval
		if cfg.Interval <= 0 || cfg.Interval > time.Hour {
			errs = append(errs, fmt.Errorf("%w: %s",
				errMonitorInvalidInterval, cfg.Interval,
			))
		}
	}

	{ // timeout
		if cfg.Timeout <= 0 {
			errs = append(errs, fmt.Errorf("%w: %s <= 0",
				errMonitorInvalidTimeout, cfg.Timeout,
			))
		}
		if cfg.Timeout > time.Minute {
			errs = append(errs, fmt.Errorf("%w: %s > 1m",
				errMonitorInvalidTimeout, cfg.Timeout,
			))
		}
		if cfg.Timeout >= cfg.Interval {
			errs = append(errs, fmt.Errorf("%w: %s >= %s",
				errMonitorInvalidTimeout, cfg.Timeout, cfg.Interval,
			))
		}
	}

	return utils.FlattenErrors(errs)
}
