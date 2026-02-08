package nethexec

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
)

// NethermindExecutionClient wraps NethRpcClient to implement Nitro's execution interfaces.
// Execution methods are always delegated to Nethermind via RPC.
// Sequencing methods are delegated to Nethermind via RPC when execEngine/sequencer are nil
// (external sequencer mode), or to internal Go components when provided (internal sequencer mode).
type NethermindExecutionClient struct {
	rpcClient  *NethRpcClient
	execEngine *gethexec.ExecutionEngine // Optional: internal execution engine for sequencing
	sequencer  *gethexec.Sequencer       // Optional: internal sequencer
}

func NewNethermindExecutionClient(url string, wsUrl string, execEngine *gethexec.ExecutionEngine, sequencer *gethexec.Sequencer) (*NethermindExecutionClient, error) {
	rpcClient, err := NewNethRpcClient(url, wsUrl)
	if err != nil {
		return nil, err
	}
	return &NethermindExecutionClient{
		rpcClient:  rpcClient,
		execEngine: execEngine,
		sequencer:  sequencer,
	}, nil
}

// DigestMessage implements ExecutionClient.DigestMessage
func (p *NethermindExecutionClient) DigestMessage(index arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	promise := containers.NewPromise[*execution.MessageResult](nil)
	go func() {
		res := p.rpcClient.DigestMessage(context.Background(), index, msg, msgForPrefetch)
		if res == nil {
			promise.ProduceError(fmt.Errorf("external DigestMessage returned nil"))
			return
		}
		promise.Produce(res)
	}()
	return &promise
}

// SetFinalityData implements ExecutionClient.SetFinalityData
func (p *NethermindExecutionClient) SetFinalityData(safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	promise := containers.NewPromise[struct{}](nil)
	go func() {
		err := p.rpcClient.SetFinalityData(context.Background(), safeFinalityData, finalizedFinalityData, validatedFinalityData)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(struct{}{})
	}()
	return &promise
}

// SetConsensusSyncData implements ExecutionClient.SetConsensusSyncData
func (p *NethermindExecutionClient) SetConsensusSyncData(syncData *execution.ConsensusSyncData) containers.PromiseInterface[struct{}] {
	promise := containers.NewPromise[struct{}](nil)
	go func() {
		err := p.rpcClient.SetConsensusSyncData(context.Background(), syncData)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(struct{}{})
	}()
	return &promise
}

// Reorg implements ExecutionClient.Reorg
// Note: main Nitro's interface has 2 parameters (no oldMessages)
func (p *NethermindExecutionClient) Reorg(msgIdxOfFirstMsgToAdd arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo) containers.PromiseInterface[[]*execution.MessageResult] {
	promise := containers.NewPromise[[]*execution.MessageResult](nil)
	go func() {
		res, err := p.rpcClient.Reorg(context.Background(), msgIdxOfFirstMsgToAdd, newMessages)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(res)
	}()
	return &promise
}

// HeadMessageIndex implements ExecutionClient.HeadMessageIndex
func (p *NethermindExecutionClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	promise := containers.NewPromise[arbutil.MessageIndex](nil)
	go func() {
		idx, err := p.rpcClient.HeadMessageIndex(context.Background())
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(idx)
	}()
	return &promise
}

// ResultAtMessageIndex implements ExecutionClient.ResultAtMessageIndex
func (p *NethermindExecutionClient) ResultAtMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	promise := containers.NewPromise[*execution.MessageResult](nil)
	go func() {
		res, err := p.rpcClient.ResultAtMessageIndex(context.Background(), msgIdx)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(res)
	}()
	return &promise
}

// MessageIndexToBlockNumber implements ExecutionClient.MessageIndexToBlockNumber
func (p *NethermindExecutionClient) MessageIndexToBlockNumber(messageIndex arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	promise := containers.NewPromise[uint64](nil)
	go func() {
		num, err := p.rpcClient.MessageIndexToBlockNumber(context.Background(), messageIndex)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(num)
	}()
	return &promise
}

// BlockNumberToMessageIndex implements ExecutionClient.BlockNumberToMessageIndex
func (p *NethermindExecutionClient) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	promise := containers.NewPromise[arbutil.MessageIndex](nil)
	go func() {
		idx, err := p.rpcClient.BlockNumberToMessageIndex(context.Background(), blockNum)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(idx)
	}()
	return &promise
}

// MarkFeedStart implements ExecutionClient.MarkFeedStart
func (p *NethermindExecutionClient) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	promise := containers.NewPromise[struct{}](nil)
	go func() {
		err := p.rpcClient.MarkFeedStart(context.Background(), to)
		if err != nil {
			promise.ProduceError(err)
			return
		}
		promise.Produce(struct{}{})
	}()
	return &promise
}

// TriggerMaintenance implements ExecutionClient.TriggerMaintenance
func (p *NethermindExecutionClient) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, fmt.Errorf("TriggerMaintenance not implemented for external execution"))
}

// ShouldTriggerMaintenance implements ExecutionClient.ShouldTriggerMaintenance
func (p *NethermindExecutionClient) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	return containers.NewReadyPromise(false, nil) // Conservative default - don't trigger maintenance
}

// MaintenanceStatus implements ExecutionClient.MaintenanceStatus
func (p *NethermindExecutionClient) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	return containers.NewReadyPromise(&execution.MaintenanceStatus{IsRunning: false}, nil)
}

// Start implements ExecutionClient.Start
func (p *NethermindExecutionClient) Start(ctx context.Context) error {
	if p.rpcClient == nil {
		return fmt.Errorf("RPC client is not initialized")
	}
	// TODO: Add a health check RPC call to verify Nethermind is accessible
	return nil
}

// StopAndWait implements ExecutionClient.StopAndWait
func (p *NethermindExecutionClient) StopAndWait() {
	if p.rpcClient != nil {
		p.rpcClient.Close()
	}
}

// Pause implements ExecutionSequencer.Pause
// Uses internal sequencer if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) Pause() {
	if p.sequencer != nil {
		p.sequencer.Pause()
		return
	}
	if err := p.rpcClient.Pause(context.Background()); err != nil {
		log.Error("Failed to call Pause on Nethermind", "error", err)
	}
}

// Activate implements ExecutionSequencer.Activate
// Uses internal sequencer if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) Activate() {
	if p.sequencer != nil {
		p.sequencer.Activate()
		return
	}
	if err := p.rpcClient.Activate(context.Background()); err != nil {
		log.Error("Failed to call Activate on Nethermind", "error", err)
	}
}

// ForwardTo implements ExecutionSequencer.ForwardTo
// Uses internal sequencer if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) ForwardTo(url string) error {
	if p.sequencer != nil {
		return p.sequencer.ForwardTo(url)
	}
	return p.rpcClient.ForwardTo(context.Background(), url)
}

// StartSequencing implements ExecutionSequencer.StartSequencing
// Uses internal sequencer if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) StartSequencing(ctx context.Context) (*execution.SequencedMsg, time.Duration) {
	if p.sequencer != nil {
		return p.sequencer.StartSequencing(ctx)
	}
	msg, dur, err := p.rpcClient.StartSequencing(ctx)
	if err != nil {
		log.Error("Failed to call StartSequencing on Nethermind", "error", err)
		return nil, 0
	}
	return msg, dur
}

// EndSequencing implements ExecutionSequencer.EndSequencing
// Uses internal sequencer if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) EndSequencing(ctx context.Context, errWhileSequencing error) {
	if p.sequencer != nil {
		p.sequencer.EndSequencing(ctx, errWhileSequencing)
		return
	}
	if err := p.rpcClient.EndSequencing(ctx, errWhileSequencing); err != nil {
		log.Error("Failed to call EndSequencing on Nethermind", "error", err)
	}
}

// EnqueueDelayedMessages implements ExecutionSequencer.EnqueueDelayedMessages
// Uses internal execution engine if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) EnqueueDelayedMessages(msgs []*arbostypes.L1IncomingMessage, firstMsgIdx uint64) {
	if p.execEngine != nil {
		p.execEngine.EnqueueDelayedMessages(msgs, firstMsgIdx)
		return
	}
	if err := p.rpcClient.EnqueueDelayedMessages(context.Background(), msgs, firstMsgIdx); err != nil {
		log.Error("Failed to call EnqueueDelayedMessages on Nethermind", "error", err)
	}
}

// AppendLastSequencedBlock implements ExecutionSequencer.AppendLastSequencedBlock
// Uses internal execution engine if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) AppendLastSequencedBlock() error {
	if p.execEngine != nil {
		return p.execEngine.AppendLastSequencedBlock()
	}
	return p.rpcClient.AppendLastSequencedBlock(context.Background())
}

// ResequenceReorgedMessage implements ExecutionSequencer.ResequenceReorgedMessage
// Uses internal execution engine if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) ResequenceReorgedMessage(msg *arbostypes.MessageWithMetadata) (*execution.SequencedMsg, error) {
	if p.execEngine != nil {
		return p.execEngine.ResequenceReorgedMessage(msg)
	}
	return p.rpcClient.ResequenceReorgedMessage(context.Background(), msg)
}

// NextDelayedMessageNumber implements ExecutionSequencer.NextDelayedMessageNumber
// Uses internal execution engine if available, otherwise delegates to Nethermind via RPC.
func (p *NethermindExecutionClient) NextDelayedMessageNumber() (uint64, error) {
	if p.execEngine != nil {
		return p.execEngine.NextDelayedMessageNumber()
	}
	return p.rpcClient.NextDelayedMessageNumber(context.Background())
}

// Synced implements ExecutionSequencer.Synced
func (p *NethermindExecutionClient) Synced(ctx context.Context) bool {
	synced, err := p.rpcClient.Synced(ctx)
	if err != nil {
		log.Error("Failed to get Synced status from Nethermind", "error", err)
		return false // Conservative default on error
	}
	return synced
}

// FullSyncProgressMap implements ExecutionSequencer.FullSyncProgressMap
func (p *NethermindExecutionClient) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	progressMap, err := p.rpcClient.FullSyncProgressMap(ctx)
	if err != nil {
		log.Error("Failed to get FullSyncProgressMap from Nethermind", "error", err)
		return map[string]interface{}{} // Empty map on error
	}
	return progressMap
}

// RecordBlockCreation implements ExecutionRecorder.RecordBlockCreation
func (p *NethermindExecutionClient) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, wasmTargets []rawdb.WasmTarget) (*execution.RecordResult, error) {
	return nil, fmt.Errorf("RecordBlockCreation not implemented for external execution")
}

// MarkValid implements ExecutionRecorder.MarkValid
func (p *NethermindExecutionClient) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
	// no-op for external execution
}

// PrepareForRecord implements ExecutionRecorder.PrepareForRecord
func (p *NethermindExecutionClient) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	return fmt.Errorf("PrepareForRecord not implemented for external execution")
}

// ArbOSVersionForMessageIndex implements ArbOSVersionGetter.ArbOSVersionForMessageIndex
func (p *NethermindExecutionClient) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](0, fmt.Errorf("ArbOSVersionForMessageIndex not implemented for external execution"))
}

// SetConsensusClient implements ExecutionNodeBridge.SetConsensusClient
func (p *NethermindExecutionClient) SetConsensusClient(consensus execution.FullConsensusClient) {
	// no-op until consensus path is implemented
}

// DigestInitMessage implements InitMessageDigester.DigestInitMessage
func (p *NethermindExecutionClient) DigestInitMessage(ctx context.Context, initialL1BaseFee *big.Int, serializedChainConfig []byte) *execution.MessageResult {
	return p.rpcClient.DigestInitMessage(ctx, initialL1BaseFee, serializedChainConfig)
}

// Initialize implements ExecutionNodeBridge.Initialize
func (p *NethermindExecutionClient) Initialize(ctx context.Context) error {
	return p.Start(ctx)
}
