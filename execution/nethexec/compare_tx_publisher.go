package nethexec

import (
	"context"

	"github.com/ethereum/go-ethereum/arbitrum_types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/timeboost"
)

var _ gethexec.TransactionPublisher = (*ComparatorTxPublisher)(nil)

// Forwarding to Nethermind is synchronous to preserve Arbitrum's FIFO ordering guarantee.
type ComparatorTxPublisher struct {
	primary    gethexec.TransactionPublisher
	nethClient *rpc.Client
}

func NewComparatorTxPublisher(primary gethexec.TransactionPublisher, nethermindUrl string) (*ComparatorTxPublisher, error) {
	nethClient, err := rpc.Dial(nethermindUrl)
	if err != nil {
		return nil, err
	}
	return &ComparatorTxPublisher{
		primary:    primary,
		nethClient: nethClient,
	}, nil
}

func (p *ComparatorTxPublisher) forwardToNethermind(ctx context.Context, tx *types.Transaction, label string) {
	rawTx, err := tx.MarshalBinary()
	if err != nil {
		log.Error("ComparatorTxPublisher: failed to encode tx for Nethermind", "label", label, "err", err)
		return
	}
	if err := p.nethClient.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(rawTx)); err != nil {
		log.Warn("ComparatorTxPublisher: Nethermind forwarding failed", "label", label, "err", err)
	}
}

func (p *ComparatorTxPublisher) PublishTransaction(ctx context.Context, tx *types.Transaction, options *arbitrum_types.ConditionalOptions) error {
	p.forwardToNethermind(ctx, tx, "PublishTransaction")
	err := p.primary.PublishTransaction(ctx, tx, options)
	return err
}

func (p *ComparatorTxPublisher) PublishExpressLaneTransaction(ctx context.Context, msg *timeboost.ExpressLaneSubmission) error {
	err := p.primary.PublishExpressLaneTransaction(ctx, msg)
	p.forwardToNethermind(ctx, msg.Transaction, "PublishExpressLaneTransaction")
	return err
}

func (p *ComparatorTxPublisher) PublishAuctionResolutionTransaction(ctx context.Context, tx *types.Transaction) error {
	err := p.primary.PublishAuctionResolutionTransaction(ctx, tx)
	p.forwardToNethermind(ctx, tx, "PublishAuctionResolutionTransaction")
	return err
}

func (p *ComparatorTxPublisher) CheckHealth(ctx context.Context) error {
	return p.primary.CheckHealth(ctx)
}

func (p *ComparatorTxPublisher) Initialize(ctx context.Context) error {
	return p.primary.Initialize(ctx)
}

func (p *ComparatorTxPublisher) Start(ctx context.Context) error {
	return p.primary.Start(ctx)
}

func (p *ComparatorTxPublisher) StopAndWait() {
	p.primary.StopAndWait()
	p.nethClient.Close()
}

func (p *ComparatorTxPublisher) Started() bool {
	return p.primary.Started()
}
