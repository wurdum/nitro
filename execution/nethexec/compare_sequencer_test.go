package nethexec

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/execution"
)

// --- Fake sequencer ---

type fakeExecSequencer struct {
	fakeExecClient

	startSeqMsg  *execution.SequencedMsg
	startSeqDur  time.Duration
	endSeqCalled bool
	endSeqErr    error

	enqueueCalled   bool
	enqueueMsgs     []*arbostypes.L1IncomingMessage
	enqueueFirstIdx uint64
	appendCalled    bool
	appendErr       error
	pauseCalled     bool
	activateCalled  bool
	forwardToCalled bool
	forwardToURL    string

	reseqMsg *execution.SequencedMsg
	reseqErr error

	nextDelayedNum uint64
	nextDelayedErr error

	synced          bool
	syncProgressMap map[string]interface{}
}

func (f *fakeExecSequencer) Pause()    { f.pauseCalled = true }
func (f *fakeExecSequencer) Activate() { f.activateCalled = true }
func (f *fakeExecSequencer) ForwardTo(url string) error {
	f.forwardToCalled = true
	f.forwardToURL = url
	return nil
}

func (f *fakeExecSequencer) StartSequencing(_ context.Context, _ execution.SequencingContext) (*execution.SequencedMsg, time.Duration) {
	return f.startSeqMsg, f.startSeqDur
}

func (f *fakeExecSequencer) EndSequencing(_ context.Context, err error) {
	f.endSeqCalled = true
	f.endSeqErr = err
}

func (f *fakeExecSequencer) EnqueueDelayedMessages(msgs []*arbostypes.L1IncomingMessage, firstMsgIdx uint64) {
	f.enqueueCalled = true
	f.enqueueMsgs = msgs
	f.enqueueFirstIdx = firstMsgIdx
}

func (f *fakeExecSequencer) AppendLastSequencedBlock() error {
	f.appendCalled = true
	return f.appendErr
}

func (f *fakeExecSequencer) ResequenceReorgedMessage(_ *arbostypes.MessageWithMetadata) (*execution.SequencedMsg, error) {
	return f.reseqMsg, f.reseqErr
}

func (f *fakeExecSequencer) NextDelayedMessageNumber() (uint64, error) {
	return f.nextDelayedNum, f.nextDelayedErr
}

func (f *fakeExecSequencer) Synced(_ context.Context) bool {
	return f.synced
}

func (f *fakeExecSequencer) FullSyncProgressMap(_ context.Context) map[string]interface{} {
	return f.syncProgressMap
}

// --- Tests ---

func TestComparatorSequencer_StartSequencing_Match(t *testing.T) {
	msg := &execution.SequencedMsg{
		MsgIdx: 42,
		MsgResult: &execution.MessageResult{
			BlockHash: common.HexToHash("0xabcd"),
			SendRoot:  common.HexToHash("0x1234"),
		},
	}
	internal := &fakeExecSequencer{startSeqMsg: msg, startSeqDur: 100 * time.Millisecond}
	external := &fakeExecSequencer{startSeqMsg: msg, startSeqDur: 50 * time.Millisecond}

	fatalChan := make(chan error, 1)
	client := NewComparatorClient(internal, external, fatalChan)
	seq := NewComparatorSequencer(client, internal, external, fatalChan)

	result, dur := seq.StartSequencing(context.Background(), execution.SequencingContext{})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.MsgIdx != 42 {
		t.Fatalf("expected MsgIdx 42, got %d", result.MsgIdx)
	}
	if dur != 50*time.Millisecond {
		t.Fatalf("expected external duration 50ms, got %v", dur)
	}

	select {
	case err := <-fatalChan:
		t.Fatalf("unexpected fatal error: %v", err)
	default:
	}
}

func TestComparatorSequencer_StartSequencing_Mismatch(t *testing.T) {
	intMsg := &execution.SequencedMsg{
		MsgIdx:    42,
		MsgResult: &execution.MessageResult{BlockHash: common.HexToHash("0xaaaa")},
	}
	extMsg := &execution.SequencedMsg{
		MsgIdx:    42,
		MsgResult: &execution.MessageResult{BlockHash: common.HexToHash("0xbbbb")},
	}
	internal := &fakeExecSequencer{startSeqMsg: intMsg}
	external := &fakeExecSequencer{startSeqMsg: extMsg}

	fatalChan := make(chan error, 1)
	client := NewComparatorClient(internal, external, fatalChan)
	seq := NewComparatorSequencer(client, internal, external, fatalChan)

	result, _ := seq.StartSequencing(context.Background(), execution.SequencingContext{})
	if result == nil {
		t.Fatal("expected non-nil result (internal)")
	}
	if result.MsgResult.BlockHash != intMsg.MsgResult.BlockHash {
		t.Fatalf("expected internal block hash, got %v", result.MsgResult.BlockHash)
	}

	select {
	case <-fatalChan:
		// Divergence reported
	default:
		t.Fatal("expected fatal error for mismatch")
	}
}

func TestComparatorSequencer_StartSequencing_BothNil(t *testing.T) {
	internal := &fakeExecSequencer{startSeqMsg: nil}
	external := &fakeExecSequencer{startSeqMsg: nil, startSeqDur: time.Second}

	fatalChan := make(chan error, 1)
	client := NewComparatorClient(internal, external, fatalChan)
	seq := NewComparatorSequencer(client, internal, external, fatalChan)

	result, dur := seq.StartSequencing(context.Background(), execution.SequencingContext{})
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
	if dur != time.Second {
		t.Fatalf("expected internal duration 1s, got %v", dur)
	}

	select {
	case err := <-fatalChan:
		t.Fatalf("unexpected fatal error: %v", err)
	default:
	}
}

func TestComparatorSequencer_EndSequencing_BothCalled(t *testing.T) {
	internal := &fakeExecSequencer{}
	external := &fakeExecSequencer{}

	client := NewComparatorClient(internal, external, nil)
	seq := NewComparatorSequencer(client, internal, external, nil)

	seq.EndSequencing(context.Background(), nil)

	if !internal.endSeqCalled {
		t.Fatal("expected internal EndSequencing to be called")
	}
	if !external.endSeqCalled {
		t.Fatal("expected external EndSequencing to be called")
	}
}

func TestComparatorSequencer_EnqueueDelayedMessages_BothCalled(t *testing.T) {
	internal := &fakeExecSequencer{}
	external := &fakeExecSequencer{}

	client := NewComparatorClient(internal, external, nil)
	seq := NewComparatorSequencer(client, internal, external, nil)

	msgs := []*arbostypes.L1IncomingMessage{{}}
	seq.EnqueueDelayedMessages(msgs, 5)

	if !internal.enqueueCalled {
		t.Fatal("expected internal EnqueueDelayedMessages to be called")
	}
	if !external.enqueueCalled {
		t.Fatal("expected external EnqueueDelayedMessages to be called")
	}
	if internal.enqueueFirstIdx != 5 {
		t.Fatalf("expected firstIdx 5, got %d", internal.enqueueFirstIdx)
	}
}

func TestComparatorSequencer_Pause_InternalOnly(t *testing.T) {
	internal := &fakeExecSequencer{}
	external := &fakeExecSequencer{}

	client := NewComparatorClient(internal, external, nil)
	seq := NewComparatorSequencer(client, internal, external, nil)

	seq.Pause()

	if !internal.pauseCalled {
		t.Fatal("expected internal Pause to be called")
	}
	if external.pauseCalled {
		t.Fatal("expected external Pause NOT to be called")
	}
}

func TestComparatorSequencer_Activate_InternalOnly(t *testing.T) {
	internal := &fakeExecSequencer{}
	external := &fakeExecSequencer{}

	client := NewComparatorClient(internal, external, nil)
	seq := NewComparatorSequencer(client, internal, external, nil)

	seq.Activate()

	if !internal.activateCalled {
		t.Fatal("expected internal Activate to be called")
	}
	if external.activateCalled {
		t.Fatal("expected external Activate NOT to be called")
	}
}

func TestComparatorSequencer_AppendLastSequencedBlock_BothCalled(t *testing.T) {
	internal := &fakeExecSequencer{}
	external := &fakeExecSequencer{}

	client := NewComparatorClient(internal, external, nil)
	seq := NewComparatorSequencer(client, internal, external, nil)

	err := seq.AppendLastSequencedBlock()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !internal.appendCalled {
		t.Fatal("expected internal AppendLastSequencedBlock to be called")
	}
	if !external.appendCalled {
		t.Fatal("expected external AppendLastSequencedBlock to be called")
	}
}

func TestComparatorSequencer_Synced_Match(t *testing.T) {
	internal := &fakeExecSequencer{synced: true}
	external := &fakeExecSequencer{synced: true}

	client := NewComparatorClient(internal, external, nil)
	seq := NewComparatorSequencer(client, internal, external, nil)

	if !seq.Synced(context.Background()) {
		t.Fatal("expected Synced to return true")
	}
}

func TestComparatorSequencer_SyncedMismatch_ReturnsInternal(t *testing.T) {
	internal := &fakeExecSequencer{synced: true}
	external := &fakeExecSequencer{synced: false}

	client := NewComparatorClient(internal, external, nil)
	seq := NewComparatorSequencer(client, internal, external, nil)

	if !seq.Synced(context.Background()) {
		t.Fatal("expected Synced to return internal value (true)")
	}
}
