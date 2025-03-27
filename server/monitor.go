package server

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/flashbots/bmonitor/jrpc"
	"github.com/flashbots/bmonitor/logutils"
	"github.com/flashbots/bmonitor/types"
	"github.com/flashbots/bmonitor/utils"
	"go.uber.org/zap"
)

func (s *Server) monitor(ctx context.Context, ts time.Time) {
	l := logutils.LoggerFromContext(ctx).With(
		zap.Int64("ts", ts.Unix()),
	)
	ctx = logutils.ContextWithLogger(ctx, l)

	l.Debug("Running new monitoring pass...")

	var (
		status = make(map[string]*types.BuilderStatus, len(s.builders))
		mx     sync.Mutex
		wg     sync.WaitGroup
	)

	for name, rpc := range s.builders {
		wg.Add(1)

		go func() {
			defer wg.Done()

			s := s.getStatus(ctx, rpc)
			mx.Lock()
			status[name] = s
			mx.Unlock()
		}()
	}

	wg.Wait()

	s.process(ctx, status)
}

func (s *Server) process(ctx context.Context, status map[string]*types.BuilderStatus) {
	s.analysePeers(ctx, status)
	s.analyseTxpool(ctx, status)
}

func (s *Server) getStatus(ctx context.Context, builder *ethclient.Client) *types.BuilderStatus {
	l := logutils.LoggerFromContext(ctx)

	res := &types.BuilderStatus{}
	errs := make([]error, 0)

	if peers, err := s.getPeers(ctx, builder); err == nil {
		res.Peers = peers
	} else {
		errs = append(errs, err)
		l.Error("Failed to get builder's peers",
			zap.Error(err),
		)
	}

	if txpool, err := s.getTxpool(ctx, builder); err == nil {
		res.Txpool = txpool
	} else {
		errs = append(errs, err)
		l.Error("Failed to get builder's txpool",
			zap.Error(err),
		)
	}

	res.Err = utils.FlattenErrors(errs)

	return res
}

func (s *Server) getPeers(ctx context.Context, builder *ethclient.Client) (*jrpc.AdminPeers, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Monitor.Timeout)
	defer cancel()

	res := &jrpc.AdminPeers{}
	if err := builder.Client().CallContext(ctx, res, "admin_peers"); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Server) getTxpool(ctx context.Context, builder *ethclient.Client) (*jrpc.TxpoolContent, error) {
	ctx, cancel := context.WithTimeout(ctx, s.cfg.Monitor.Timeout)
	defer cancel()

	res := &jrpc.TxpoolContent{}
	if err := builder.Client().CallContext(ctx, res, "txpool_content"); err != nil {
		return nil, err
	}

	return res, nil
}
