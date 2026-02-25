package nethexec

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution"
)

var _ execution.ExecutionSequencer = (*ComparatorSequencer)(nil)

// Geth is canonical; divergences are reported via fatalErrChan.
type ComparatorSequencer struct {
	*ComparatorClient
	internalSeq  execution.ExecutionSequencer
	externalSeq  execution.ExecutionSequencer
	fatalErrChan chan error
}

func NewComparatorSequencer(
	comparatorClient *ComparatorClient,
	internalSeq execution.ExecutionSequencer,
	externalSeq execution.ExecutionSequencer,
	fatalErrChan chan error,
) *ComparatorSequencer {
	return &ComparatorSequencer{
		ComparatorClient: comparatorClient,
		internalSeq:      internalSeq,
		externalSeq:      externalSeq,
		fatalErrChan:     fatalErrChan,
	}
}

// --- Compared operations ---

func (s *ComparatorSequencer) StartSequencing(ctx context.Context) (*execution.SequencedMsg, time.Duration) {
	intMsg, _ := s.internalSeq.StartSequencing(ctx)
	extMsg, extDur := s.externalSeq.StartSequencing(ctx)

	if intMsg == nil && extMsg == nil {
		return nil, extDur
	}

	if err := compare("StartSequencing", intMsg, nil, extMsg, nil); err != nil && s.fatalErrChan != nil {
		reportDivergence(s.fatalErrChan, "StartSequencing", err)
	}

	return intMsg, extDur
}

func (s *ComparatorSequencer) ResequenceReorgedMessage(msg *arbostypes.MessageWithMetadata) (*execution.SequencedMsg, error) {
	intRes, intErr := s.internalSeq.ResequenceReorgedMessage(msg)
	extRes, extErr := s.externalSeq.ResequenceReorgedMessage(msg)

	if err := compare("ResequenceReorgedMessage", intRes, intErr, extRes, extErr); err != nil && s.fatalErrChan != nil {
		reportDivergence(s.fatalErrChan, "ResequenceReorgedMessage", err)
	}

	return intRes, intErr
}

func (s *ComparatorSequencer) NextDelayedMessageNumber() (uint64, error) {
	intRes, intErr := s.internalSeq.NextDelayedMessageNumber()
	extRes, extErr := s.externalSeq.NextDelayedMessageNumber()

	if err := compare("NextDelayedMessageNumber", intRes, intErr, extRes, extErr); err != nil && s.fatalErrChan != nil {
		reportDivergence(s.fatalErrChan, "NextDelayedMessageNumber", err)
	}

	return intRes, intErr
}

// --- Mirrored operations ---

func (s *ComparatorSequencer) EndSequencing(ctx context.Context, errWhileSequencing error) {
	s.internalSeq.EndSequencing(ctx, errWhileSequencing)
	s.externalSeq.EndSequencing(ctx, errWhileSequencing)
}

func (s *ComparatorSequencer) AppendLastSequencedBlock() error {
	if err := s.externalSeq.AppendLastSequencedBlock(); err != nil {
		log.Warn("External AppendLastSequencedBlock failed", "err", err)
	}
	return s.internalSeq.AppendLastSequencedBlock()
}

func (s *ComparatorSequencer) EnqueueDelayedMessages(msgs []*arbostypes.L1IncomingMessage, firstMsgIdx uint64) {
	s.internalSeq.EnqueueDelayedMessages(msgs, firstMsgIdx)
	s.externalSeq.EnqueueDelayedMessages(msgs, firstMsgIdx)
}

// --- Control operations (Geth only) ---

func (s *ComparatorSequencer) Pause() {
	s.internalSeq.Pause()
}

func (s *ComparatorSequencer) Activate() {
	s.internalSeq.Activate()
}

func (s *ComparatorSequencer) ForwardTo(url string) error {
	return s.internalSeq.ForwardTo(url)
}

// --- Status operations ---

func (s *ComparatorSequencer) Synced(ctx context.Context) bool {
	intSynced := s.internalSeq.Synced(ctx)
	extSynced := s.externalSeq.Synced(ctx)
	if intSynced != extSynced {
		log.Warn("Synced status mismatch", "internal", intSynced, "external", extSynced)
	}
	return intSynced
}

func (s *ComparatorSequencer) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	return s.internalSeq.FullSyncProgressMap(ctx)
}
