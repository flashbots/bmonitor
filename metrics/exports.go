package metrics

import (
	otelapi "go.opentelemetry.io/otel/metric"
)

var (
	PeersCount                otelapi.Int64Gauge
	TxpoolDuplicateNonceCount otelapi.Int64Counter
	TxpoolNonceGapsLength     otelapi.Int64Gauge
	TxpoolMissingTxCount      otelapi.Int64Gauge
)
