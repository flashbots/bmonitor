package metrics

import (
	"context"

	"go.opentelemetry.io/otel/exporters/prometheus"
	otelapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

const (
	metricsNamespace = "bmonitor"
)

var (
	meter otelapi.Meter
)

func Setup(ctx context.Context) error {
	for _, setup := range []func(context.Context) error{
		setupMeter, // must come first
		setupPeersCount,
		setupTxpoolDuplicateNonceCount,
		setupTxpoolNonceGapsLength,
		setupTxpoolMissingTxCount,
	} {
		if err := setup(ctx); err != nil {
			return err
		}
	}

	return nil
}

func setupMeter(ctx context.Context) error {
	res, err := resource.New(ctx)
	if err != nil {
		return err
	}

	exporter, err := prometheus.New(
		prometheus.WithNamespace(metricsNamespace),
		prometheus.WithoutScopeInfo(),
	)
	if err != nil {
		return err
	}

	provider := metric.NewMeterProvider(
		metric.WithReader(exporter),
		metric.WithResource(res),
	)

	meter = provider.Meter(metricsNamespace)

	return nil
}

func setupPeersCount(ctx context.Context) error {
	m, err := meter.Int64Gauge("peers_count",
		otelapi.WithDescription("count of connected peers"),
	)
	if err != nil {
		return err
	}
	PeersCount = m
	return nil
}

func setupTxpoolDuplicateNonceCount(ctx context.Context) error {
	m, err := meter.Int64Counter("txpool_duplicate_nonce_count",
		otelapi.WithDescription("count of transactions seen that have same address and nonce but different hashes"),
	)
	if err != nil {
		return err
	}
	TxpoolDuplicateNonceCount = m
	return nil
}

func setupTxpoolNonceGapsLength(ctx context.Context) error {
	m, err := meter.Int64Gauge("txpool_nonce_gap_length",
		otelapi.WithDescription("cumulative length of nonce gaps"),
	)
	if err != nil {
		return err
	}
	TxpoolNonceGapsLength = m
	return nil
}

func setupTxpoolMissingTxCount(ctx context.Context) error {
	m, err := meter.Int64Gauge("txpool_missing_tx_count",
		otelapi.WithDescription("count missing transaction in the txpool"),
	)
	if err != nil {
		return err
	}
	TxpoolMissingTxCount = m
	return nil
}
