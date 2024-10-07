package osmocexfiller

import (
	"context"

	"github.com/osmosis-labs/sqs/domain"
	"github.com/osmosis-labs/sqs/domain/mvc"
	"github.com/osmosis-labs/sqs/ingest/usecase/plugins/osmocex-filler/cex"
	"go.opentelemetry.io/otel"
)

type osmocexFillerIngestPlugin struct {
	poolsUseCase  mvc.PoolsUsecase
	tokensUseCase mvc.TokensUsecase

	CExchanges []cex.CExchangeI
}

const (
	tracerName = "sqs-osmocex-filler"
)

var (
	tracer = otel.Tracer(tracerName)
)

var _ domain.EndBlockProcessPlugin = &osmocexFillerIngestPlugin{}

// func NewOsmoCexFillerIngestPlugin(poolsUseCase mvc.PoolsUsecase, tokensUseCase mvc.TokensUsecase) *osmocexFillerIngestPlugin {}

func (oc *osmocexFillerIngestPlugin) ProcessEndBlock(ctx context.Context, blockHeight uint64, metadata domain.BlockPoolMetadata) error {
	ctx, span := tracer.Start(ctx, "osmocexfiller.ProcessEndBlock")
	defer span.End()

	return nil
}
