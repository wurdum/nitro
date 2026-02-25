package nethexec

import (
	"context"
	"fmt"

	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

var (
	_ execution.ExecutionClient   = (*ComparatorClient)(nil)
	_ arbnode.ExecutionNodeBridge = (*ComparatorClient)(nil)
)

// ComparatorClient validates Nethermind against Geth by comparing execution results.
// Geth is canonical because Nethermind is not yet validated for production.
type ComparatorClient struct {
	internal     execution.ExecutionClient
	external     execution.ExecutionClient
	fatalErrChan chan error
}

func NewComparatorClient(internal, external execution.ExecutionClient, fatalErrChan chan error) *ComparatorClient {
	return &ComparatorClient{
		internal:     internal,
		external:     external,
		fatalErrChan: fatalErrChan,
	}
}

// --- Compared operations ---

func (c *ComparatorClient) DigestMessage(idx arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, prefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return comparePromises(c.fatalErrChan, "DigestMessage",
		c.internal.DigestMessage(idx, msg, prefetch),
		c.external.DigestMessage(idx, msg, prefetch),
	)
}

func (c *ComparatorClient) Reorg(msgIdx arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo) containers.PromiseInterface[[]*execution.MessageResult] {
	return comparePromises(c.fatalErrChan, "Reorg",
		c.internal.Reorg(msgIdx, newMessages),
		c.external.Reorg(msgIdx, newMessages),
	)
}

func (c *ComparatorClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	return comparePromises[arbutil.MessageIndex](nil, "HeadMessageIndex",
		c.internal.HeadMessageIndex(),
		c.external.HeadMessageIndex(),
	)
}

func (c *ComparatorClient) ResultAtMessageIndex(idx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return comparePromises[*execution.MessageResult](nil, "ResultAtMessageIndex",
		c.internal.ResultAtMessageIndex(idx),
		c.external.ResultAtMessageIndex(idx),
	)
}

func (c *ComparatorClient) MessageIndexToBlockNumber(idx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return comparePromises(c.fatalErrChan, "MessageIndexToBlockNumber",
		c.internal.MessageIndexToBlockNumber(idx),
		c.external.MessageIndexToBlockNumber(idx),
	)
}

func (c *ComparatorClient) BlockNumberToMessageIndex(num uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	return comparePromises(c.fatalErrChan, "BlockNumberToMessageIndex",
		c.internal.BlockNumberToMessageIndex(num),
		c.external.BlockNumberToMessageIndex(num),
	)
}

func (c *ComparatorClient) SetFinalityData(safe, finalized, validated *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	return comparePromises(c.fatalErrChan, "SetFinalityData",
		c.internal.SetFinalityData(safe, finalized, validated),
		c.external.SetFinalityData(safe, finalized, validated),
	)
}

func (c *ComparatorClient) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return comparePromises(c.fatalErrChan, "MarkFeedStart",
		c.internal.MarkFeedStart(to),
		c.external.MarkFeedStart(to),
	)
}

func (c *ComparatorClient) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	return comparePromises(c.fatalErrChan, "ShouldTriggerMaintenance",
		c.internal.ShouldTriggerMaintenance(),
		c.external.ShouldTriggerMaintenance(),
	)
}

func (c *ComparatorClient) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	return comparePromises(c.fatalErrChan, "MaintenanceStatus",
		c.internal.MaintenanceStatus(),
		c.external.MaintenanceStatus(),
	)
}

// --- Push-only operations ---

func (c *ComparatorClient) SetConsensusSyncData(data *execution.ConsensusSyncData) containers.PromiseInterface[struct{}] {
	c.external.SetConsensusSyncData(data)
	return c.internal.SetConsensusSyncData(data)
}

func (c *ComparatorClient) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	c.external.TriggerMaintenance()
	return c.internal.TriggerMaintenance()
}

// --- ArbOSVersionGetter ---

func (c *ComparatorClient) ArbOSVersionForMessageIndex(idx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return c.internal.ArbOSVersionForMessageIndex(idx)
}

// --- Lifecycle ---

func (c *ComparatorClient) Start(ctx context.Context) error {
	if err := c.internal.Start(ctx); err != nil {
		return fmt.Errorf("failed to start internal execution client: %w", err)
	}
	if err := c.external.Start(ctx); err != nil {
		return fmt.Errorf("failed to start external execution client: %w", err)
	}
	return nil
}

func (c *ComparatorClient) StopAndWait() {
	c.external.StopAndWait()
	c.internal.StopAndWait()
}

// --- ExecutionNodeBridge ---

func (c *ComparatorClient) Initialize(ctx context.Context) error {
	if bridge, ok := c.internal.(arbnode.ExecutionNodeBridge); ok {
		if err := bridge.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize internal execution client: %w", err)
		}
	}
	if bridge, ok := c.external.(arbnode.ExecutionNodeBridge); ok {
		if err := bridge.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize external execution client: %w", err)
		}
	}
	return nil
}

func (c *ComparatorClient) SetConsensusClient(consensus execution.FullConsensusClient) {
	if bridge, ok := c.internal.(arbnode.ExecutionNodeBridge); ok {
		bridge.SetConsensusClient(consensus)
	}
	if bridge, ok := c.external.(arbnode.ExecutionNodeBridge); ok {
		bridge.SetConsensusClient(consensus)
	}
}
