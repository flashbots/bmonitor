package server

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/flashbots/bmonitor/jrpc"
	"github.com/flashbots/bmonitor/logutils"
	"github.com/flashbots/bmonitor/metrics"
	"github.com/flashbots/bmonitor/types"
	"github.com/flashbots/bmonitor/utils"

	"go.opentelemetry.io/otel/attribute"
	otelapi "go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

func (s *Server) analysePeers(ctx context.Context, status map[string]*types.BuilderStatus) {
	l := logutils.LoggerFromContext(ctx)

	for builder, s := range status {
		if s.Peers == nil {
			continue
		}

		var loopback, internal, external int64
		for _, peer := range *s.Peers {
			addr, err := net.ResolveTCPAddr("tcp", peer.Network.RemoteAddress)
			if err != nil {
				l.Warn("Failed to parse peer's remote address",
					zap.Error(err),
					zap.String("builder", builder),
					zap.String("peer_id", peer.ID),
					zap.String("peer_enode", peer.Enode),
					zap.String("peer_ip", peer.Network.RemoteAddress),
				)
				continue
			}
			switch {
			case addr.IP.IsLoopback():
				loopback++
			case addr.IP.IsPrivate():
				internal++
			default:
				external++
				l.Debug("Builder has external peer",
					zap.String("builder", builder),
					zap.String("peer_enode", peer.Enode),
					zap.String("peer_id", peer.ID),
					zap.Strings("peer_caps", peer.Capabilities),
					zap.String("peer_name", peer.Name),
					zap.String("peer_ip", peer.Network.RemoteAddress),
				)
			}
		}

		metrics.PeersCount.Record(ctx, loopback, otelapi.WithAttributes(
			attribute.KeyValue{Key: "builder", Value: attribute.StringValue(builder)},
			attribute.KeyValue{Key: "type", Value: attribute.StringValue("loopback")},
		))

		metrics.PeersCount.Record(ctx, internal, otelapi.WithAttributes(
			attribute.KeyValue{Key: "builder", Value: attribute.StringValue(builder)},
			attribute.KeyValue{Key: "type", Value: attribute.StringValue("internal")},
		))

		metrics.PeersCount.Record(ctx, external, otelapi.WithAttributes(
			attribute.KeyValue{Key: "builder", Value: attribute.StringValue(builder)},
			attribute.KeyValue{Key: "type", Value: attribute.StringValue("external")},
		))
	}
}

func (s *Server) analyseTxpool(ctx context.Context, status map[string]*types.BuilderStatus) {
	l := logutils.LoggerFromContext(ctx)

	size := 0
	for _, sts := range status {
		if sts.Txpool == nil {
			continue
		}
		candidate := len(sts.Txpool.Pending) + len(sts.Txpool.Queued)
		if candidate > size {
			size = candidate
		}
	}

	txpoolByHash := make(map[string]*jrpc.TxpoolContent_Tx, size)
	txpoolByAddrNonce := make(map[string]map[uint64]*jrpc.TxpoolContent_Tx, size)
	nonceMin := make(map[string]uint64)
	nonceMax := make(map[string]uint64)
	addresses := make(map[string]struct{})

	ingestTx := func(tx *jrpc.TxpoolContent_Tx, builder string) {
		if _, known := addresses[tx.From]; !known {
			addresses[tx.From] = struct{}{}
		}

		if _, known := txpoolByHash[tx.Hash]; !known {
			txpoolByHash[tx.Hash] = tx
		}

		if _, known := txpoolByAddrNonce[tx.From]; !known {
			txpoolByAddrNonce[tx.From] = make(map[uint64]*jrpc.TxpoolContent_Tx)
		}
		txpoolByNonce := txpoolByAddrNonce[tx.From]

		nonce, err := strconv.ParseUint(strings.TrimPrefix(tx.Nonce, "0x"), 16, 64)
		if err != nil {
			l.Warn("Failed to parse nonce from hex into uint",
				zap.Error(err),
				zap.String("nonce", tx.Nonce),
				zap.String("builder", builder),
			)
			return
		}

		if knownTx, known := txpoolByNonce[nonce]; !known {
			txpoolByNonce[nonce] = tx
		} else if knownTx.Hash != tx.Hash {
			l.Warn("Multiple tx from same address and nonce",
				zap.String("from", tx.From),
				zap.String("known_tx_hash", knownTx.Hash),
				zap.String("other_tx_hash", tx.Hash),
				zap.String("builder", builder),
			)
			metrics.TxpoolDuplicateNonceCount.Add(ctx, 1, otelapi.WithAttributes(
				attribute.KeyValue{Key: "from", Value: attribute.StringValue(tx.From)},
			))
			return
		}

		if _, known := nonceMin[tx.From]; !known {
			nonceMin[tx.From] = nonce
		}
		nonceMin[tx.From] = min(nonce, nonceMin[tx.From])

		if _, known := nonceMax[tx.From]; !known {
			nonceMax[tx.From] = nonce
		}
		nonceMax[tx.From] = max(nonce, nonceMax[tx.From])
	}

	{ // warm up the data
		for builder, s := range status {
			if s.Txpool == nil {
				continue
			}
			for _, nonces := range s.Txpool.Pending {
				for _, tx := range nonces {
					ingestTx(tx, builder)
				}
			}
			for _, nonces := range s.Txpool.Queued {
				for _, tx := range nonces {
					ingestTx(tx, builder)
				}
			}
		}
	}

	l.Debug("Merged the txpools",
		zap.Int("size", len(txpoolByHash)),
	)

	for builder, sts := range status {
		if sts.Txpool == nil {
			continue
		}

		l.Debug("Inspecting builder's txpool...",
			zap.String("builder", builder),
			zap.Int("pending", len(sts.Txpool.Pending)),
			zap.Int("queued", len(sts.Txpool.Queued)),
		)

		missingTxCount := int64(0)
		nonceGapsLength := uint64(0)

		for addr := range addresses {
			pending := sts.Txpool.Pending[addr]
			queued := sts.Txpool.Queued[addr]

			addrEth, err := utils.ParseAddress(addr)
			if err != nil {
				l.Warn("Failed to parse a tx from address",
					zap.Error(err),
					zap.String("addr", addr),
					zap.String("builder", builder),
				)
				continue
			}

			noncePending, err := s.builders[builder].PendingNonceAt(ctx, addrEth)
			if err != nil {
				l.Warn("Failed to get pending nonce",
					zap.Error(err),
					zap.String("addr", addr),
					zap.String("builder", builder),
				)
				continue
			}

			_nonceMin := max(nonceMin[addr], noncePending)
			_nonceMax := nonceMax[addr]

			if _nonceMin > _nonceMax {
				l.Info("No un-included transactions from address, skipping",
					zap.String("builder", builder),
					zap.String("from", addr),
					zap.Uint64("nonce", noncePending),
				)
				continue
			}

			l.Info("Iterating through nonces",
				zap.String("builder", builder),
				zap.String("from", addr),
				zap.Uint64("nonce_min", _nonceMin),
				zap.Uint64("nonce_max", _nonceMax),
			)

			nonceGapStart := uint64(0)
			for nonce := _nonceMin; nonce <= _nonceMax; nonce++ {
				strNonce := strconv.FormatUint(nonce, 10)
				pendingTx, isPending := pending[strNonce]
				queuedTx, isQueued := queued[strNonce]
				tx := txpoolByAddrNonce[addr][nonce]

				switch {

				case isPending == !isQueued:
					if nonceGapStart != 0 {
						length := nonce - nonceGapStart
						nonceGapsLength += length
						l.Warn("Nonce gap detected",
							zap.String("builder", builder),
							zap.String("from", addr),
							zap.Uint64("nonce_gap_start", nonceGapStart),
							zap.Uint64("nonce_gap_end", nonce-1),
							zap.Uint64("nonce_gap_length", length),
						)
					}
					continue

				case isPending && isQueued:
					l.Warn("Same tx is both pending and queued (should never be the case)",
						zap.String("builder", builder),
						zap.String("pending_tx_hash", pendingTx.Hash),
						zap.String("queued_tx_hash", queuedTx.Hash),
					)
					continue

				default:
					if nonceGapStart == 0 {
						nonceGapStart = nonce
					}
					missingTxCount++

					if tx == nil {
						l.Warn("Tx is not known to any builder",
							zap.String("builder", builder),
							zap.String("from", addr),
							zap.String("nonce", strNonce),
						)
						continue
					}

					l.Warn("Tx is not known to the builder",
						zap.String("builder", builder),
						zap.String("from", addr),
						zap.String("nonce", strNonce),
						zap.String("tx_hash", tx.Hash),
					)
				}
			}
		}

		metrics.TxpoolNonceGapsLength.Record(ctx, int64(nonceGapsLength), otelapi.WithAttributes(
			attribute.KeyValue{Key: "builder", Value: attribute.StringValue(builder)},
		))

		metrics.TxpoolMissingTxCount.Record(ctx, missingTxCount, otelapi.WithAttributes(
			attribute.KeyValue{Key: "builder", Value: attribute.StringValue(builder)},
		))
	}
}
