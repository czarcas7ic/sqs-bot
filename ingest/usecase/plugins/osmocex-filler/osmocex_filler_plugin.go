package osmocexfiller

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/osmosis-labs/sqs/domain"
	"github.com/osmosis-labs/sqs/log"
	"golang.org/x/sync/errgroup"

	"github.com/osmosis-labs/sqs/domain/keyring"
	"github.com/osmosis-labs/sqs/domain/mvc"
	orderbookplugindomain "github.com/osmosis-labs/sqs/domain/orderbook/plugin"
	passthroughdomain "github.com/osmosis-labs/sqs/domain/passthrough"

	"github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/bybit"
	osmocexfillertypes "github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/types"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

type osmocexFillerIngestPlugin struct {
	ctx context.Context

	poolsUseCase mvc.PoolsUsecase
	// tokensUseCase mvc.TokensUsecase

	orderMapByPoolID *sync.Map

	atomicBool           atomic.Bool
	orderbookCWAPIClient orderbookplugindomain.OrderbookCWAPIClient

	Exchanges []osmocexfillertypes.ExchangeI

	logger log.Logger
}

const (
	tracerName = "sqs-osmocex-filler"
)

var (
	tracer = otel.Tracer(tracerName)
)

var _ domain.EndBlockProcessPlugin = &osmocexFillerIngestPlugin{}

func New(poolsUseCase mvc.PoolsUsecase, tokensUseCase mvc.TokensUsecase, routerUsecase mvc.RouterUsecase, passthroughGRPCClient passthroughdomain.PassthroughGRPCClient, osmoKeyring keyring.Keyring, orderbookCWAPIClient orderbookplugindomain.OrderbookCWAPIClient, logger log.Logger) *osmocexFillerIngestPlugin {
	ctx := context.Background()
	orderMapByPoolID := &sync.Map{}

	exchanges := []osmocexfillertypes.ExchangeI{
		bybit.New(ctx, orderMapByPoolID, &poolsUseCase, &tokensUseCase, &routerUsecase, &osmoKeyring, &passthroughGRPCClient, logger),
		// add other exchanges here after implementation
	}

	plugin := &osmocexFillerIngestPlugin{
		ctx:                  ctx,
		poolsUseCase:         poolsUseCase,
		orderbookCWAPIClient: orderbookCWAPIClient,
		orderMapByPoolID:     orderMapByPoolID,
		Exchanges:            make([]osmocexfillertypes.ExchangeI, 0),
		logger:               logger,
	}

	plugin.registerExchanges(exchanges)

	return plugin
}

func (oc *osmocexFillerIngestPlugin) ProcessEndBlock(ctx context.Context, blockHeight uint64, metadata domain.BlockPoolMetadata) error {
	ctx, span := tracer.Start(ctx, "osmocexfiller.ProcessEndBlock")
	defer span.End()

	canonicalOrderbooks, err := oc.poolsUseCase.GetAllCanonicalOrderbookPoolIDs()
	if err != nil {
		oc.logger.Error("failed to get all canonical orderbook pool IDs", zap.Error(err))
		return err
	}

	// Fetch ticks for all the orderbooks
	oc.fetchTicksForModifiedOrderbooks(ctx, &metadata.PoolIDs, canonicalOrderbooks)

	// For simplicity, we allow only one block to be processed at a time.
	// This may be relaxed in the future.
	if !oc.atomicBool.CompareAndSwap(false, true) {
		oc.logger.Info("osmocex filler is already in progress", zap.Uint64("block_height", blockHeight))
		return nil
	}
	defer oc.atomicBool.Store(false)

	for _, cExchange := range oc.Exchanges {
		cExchange.Signal()
	}

	return nil
}

// fetchTicksForModifiedOrderbooks fetches updated ticks for pools updated in the last block concurrently per each modified pool
func (oc *osmocexFillerIngestPlugin) fetchTicksForModifiedOrderbooks(ctx context.Context, blockUpdatedPools *map[uint64]struct{}, canonicalOrderbooks []domain.CanonicalOrderBooksResult) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, canonicalOrderbookResult := range canonicalOrderbooks {
		if _, ok := (*blockUpdatedPools)[canonicalOrderbookResult.PoolID]; ok {
			orderbookResult := canonicalOrderbookResult // Create local copy to avoid closure issues
			g.Go(func() error {
				// Fetch ticks and return error if it occurs
				if err := oc.fetchTicksForOrderbook(ctx, orderbookResult); err != nil {
					oc.logger.Error("failed to fetch ticks for orderbook", zap.Error(err), zap.Uint64("orderbook_id", orderbookResult.PoolID))
					return err
				}
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

func (oc *osmocexFillerIngestPlugin) registerExchanges(exchanges []osmocexfillertypes.ExchangeI) {
	oc.Exchanges = append(oc.Exchanges, exchanges...)

	for _, exchange := range exchanges {
		err := exchange.RegisterPairs(oc.ctx)
		if err != nil {
			oc.logger.Error("failed to register pairs", zap.Error(err))
			panic(err)
		}
	}
}
