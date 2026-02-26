package nethexec

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

// NethermindSequencerClient implements ExecutionSequencer by delegating all sequencing
// methods to Nethermind via RPC. Used in external-sequencer mode where Nethermind handles
// both execution and sequencing. Synced and FullSyncProgressMap are inherited from
// the embedded NethermindExecutionClient.
type NethermindSequencerClient struct {
	*NethermindExecutionClient
}

func NewNethermindSequencerClient(execClient *NethermindExecutionClient) *NethermindSequencerClient {
	return &NethermindSequencerClient{NethermindExecutionClient: execClient}
}

func (s *NethermindSequencerClient) Pause() {
	if err := s.rpcClient.Pause(context.Background()); err != nil {
		log.Error("Failed to call Pause on Nethermind", "error", err)
	}
}

func (s *NethermindSequencerClient) Activate() {
	if err := s.rpcClient.Activate(context.Background()); err != nil {
		log.Error("Failed to call Activate on Nethermind", "error", err)
	}
}

func (s *NethermindSequencerClient) ForwardTo(url string) error {
	return s.rpcClient.ForwardTo(context.Background(), url)
}

func (s *NethermindSequencerClient) StartSequencing(ctx context.Context, seqCtx execution.SequencingContext) (*execution.SequencedMsg, time.Duration) {
	msg, dur, err := s.rpcClient.StartSequencing(ctx, seqCtx)
	if err != nil {
		log.Error("Failed to call StartSequencing on Nethermind", "error", err)
		return nil, 0
	}
	return msg, dur
}

func (s *NethermindSequencerClient) EndSequencing(ctx context.Context, errWhileSequencing error) {
	if err := s.rpcClient.EndSequencing(ctx, errWhileSequencing); err != nil {
		log.Error("Failed to call EndSequencing on Nethermind", "error", err)
	}
}

func (s *NethermindSequencerClient) EnqueueDelayedMessages(msgs []*arbostypes.L1IncomingMessage, firstMsgIdx uint64) {
	if err := s.rpcClient.EnqueueDelayedMessages(context.Background(), msgs, firstMsgIdx); err != nil {
		log.Error("Failed to call EnqueueDelayedMessages on Nethermind", "error", err)
	}
}

func (s *NethermindSequencerClient) AppendLastSequencedBlock() error {
	return s.rpcClient.AppendLastSequencedBlock(context.Background())
}

func (s *NethermindSequencerClient) ResequenceReorgedMessage(msg *arbostypes.MessageWithMetadata) (*execution.SequencedMsg, error) {
	return s.rpcClient.ResequenceReorgedMessage(context.Background(), msg)
}

func (s *NethermindSequencerClient) NextDelayedMessageNumber() (uint64, error) {
	return s.rpcClient.NextDelayedMessageNumber(context.Background())
}

// InternalSequencerWrapper implements ExecutionSequencer for external-execution mode
// when Sequencer.Enable=true. ExecutionClient methods are inherited from NethermindExecutionClient
// (delegated to Nethermind via RPC), while sequencing methods use internal Geth components.
// Synced and FullSyncProgressMap are inherited from the embedded NethermindExecutionClient.
type InternalSequencerWrapper struct {
	*NethermindExecutionClient
	execEngine *gethexec.ExecutionEngine
	sequencer  *gethexec.Sequencer
}

func NewInternalSequencerWrapper(execClient *NethermindExecutionClient, execEngine *gethexec.ExecutionEngine, sequencer *gethexec.Sequencer) *InternalSequencerWrapper {
	return &InternalSequencerWrapper{
		NethermindExecutionClient: execClient,
		execEngine:                execEngine,
		sequencer:                 sequencer,
	}
}

func (w *InternalSequencerWrapper) Pause() {
	w.sequencer.Pause()
}

func (w *InternalSequencerWrapper) Activate() {
	w.sequencer.Activate()
}

func (w *InternalSequencerWrapper) ForwardTo(url string) error {
	return w.sequencer.ForwardTo(url)
}

func (w *InternalSequencerWrapper) StartSequencing(ctx context.Context, seqCtx execution.SequencingContext) (*execution.SequencedMsg, time.Duration) {
	return w.sequencer.StartSequencing(ctx, seqCtx)
}

func (w *InternalSequencerWrapper) EndSequencing(ctx context.Context, errWhileSequencing error) {
	w.sequencer.EndSequencing(ctx, errWhileSequencing)
}

func (w *InternalSequencerWrapper) EnqueueDelayedMessages(msgs []*arbostypes.L1IncomingMessage, firstMsgIdx uint64) {
	w.execEngine.EnqueueDelayedMessages(msgs, firstMsgIdx)
}

func (w *InternalSequencerWrapper) AppendLastSequencedBlock() error {
	return w.execEngine.AppendLastSequencedBlock()
}

func (w *InternalSequencerWrapper) ResequenceReorgedMessage(msg *arbostypes.MessageWithMetadata) (*execution.SequencedMsg, error) {
	return w.execEngine.ResequenceReorgedMessage(msg)
}

func (w *InternalSequencerWrapper) NextDelayedMessageNumber() (uint64, error) {
	return w.execEngine.NextDelayedMessageNumber()
}
