package nethexec

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

// --- Fake types ---

// fakeExecClient implements execution.ExecutionClient with configurable return values.
type fakeExecClient struct {
	digestResult *execution.MessageResult
	digestErr    error
	reorgResults []*execution.MessageResult
	reorgErr     error
	startErr     error
	startCalled  bool
	stopCalled   bool
}

func (m *fakeExecClient) DigestMessage(_ arbutil.MessageIndex, _ *arbostypes.MessageWithMetadata, _ *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	return containers.NewReadyPromise(m.digestResult, m.digestErr)
}

func (m *fakeExecClient) Reorg(_ arbutil.MessageIndex, _ []arbostypes.MessageWithMetadataAndBlockInfo) containers.PromiseInterface[[]*execution.MessageResult] {
	return containers.NewReadyPromise(m.reorgResults, m.reorgErr)
}

func (m *fakeExecClient) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise[arbutil.MessageIndex](0, nil)
}

func (m *fakeExecClient) ResultAtMessageIndex(_ arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	return containers.NewReadyPromise(m.digestResult, nil)
}

func (m *fakeExecClient) MessageIndexToBlockNumber(_ arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](0, nil)
}

func (m *fakeExecClient) BlockNumberToMessageIndex(_ uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	return containers.NewReadyPromise[arbutil.MessageIndex](0, nil)
}

func (m *fakeExecClient) SetFinalityData(_ *arbutil.FinalityData, _ *arbutil.FinalityData, _ *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (m *fakeExecClient) SetConsensusSyncData(_ *execution.ConsensusSyncData) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (m *fakeExecClient) MarkFeedStart(_ arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (m *fakeExecClient) TriggerMaintenance() containers.PromiseInterface[struct{}] {
	return containers.NewReadyPromise(struct{}{}, nil)
}

func (m *fakeExecClient) ShouldTriggerMaintenance() containers.PromiseInterface[bool] {
	return containers.NewReadyPromise(false, nil)
}

func (m *fakeExecClient) MaintenanceStatus() containers.PromiseInterface[*execution.MaintenanceStatus] {
	return containers.NewReadyPromise(&execution.MaintenanceStatus{IsRunning: false}, nil)
}

func (m *fakeExecClient) ArbOSVersionForMessageIndex(_ arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	return containers.NewReadyPromise[uint64](0, nil)
}

func (m *fakeExecClient) Start(_ context.Context) error {
	m.startCalled = true
	return m.startErr
}

func (m *fakeExecClient) StopAndWait() {
	m.stopCalled = true
}

// fakeBridgeClient extends fakeExecClient with ExecutionNodeBridge support.
type fakeBridgeClient struct {
	fakeExecClient
	initializeCalled   bool
	initializeErr      error
	setConsensusCalled bool
}

func (m *fakeBridgeClient) Initialize(_ context.Context) error {
	m.initializeCalled = true
	return m.initializeErr
}

func (m *fakeBridgeClient) SetConsensusClient(_ execution.FullConsensusClient) {
	m.setConsensusCalled = true
}

// --- Tests ---

func TestComparatorClient_DigestMessage_Match(t *testing.T) {
	result := &execution.MessageResult{
		BlockHash: common.HexToHash("0x1234"),
		SendRoot:  common.HexToHash("0x5678"),
	}
	internal := &fakeExecClient{digestResult: result}
	external := &fakeExecClient{digestResult: result}

	c := NewComparatorClient(internal, external, make(chan error, 1))
	promise := c.DigestMessage(0, nil, nil)

	select {
	case <-promise.ReadyChan():
		res, err := promise.Current()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if res.BlockHash != result.BlockHash {
			t.Fatalf("expected block hash %v, got %v", result.BlockHash, res.BlockHash)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("promise did not resolve within timeout")
	}
}

func TestComparatorClient_DigestMessage_Mismatch(t *testing.T) {
	intRes := &execution.MessageResult{BlockHash: common.HexToHash("0x1234")}
	extRes := &execution.MessageResult{BlockHash: common.HexToHash("0xaaaa")}
	internal := &fakeExecClient{digestResult: intRes}
	external := &fakeExecClient{digestResult: extRes}

	fatalChan := make(chan error, 1)
	c := NewComparatorClient(internal, external, fatalChan)
	promise := c.DigestMessage(0, nil, nil)

	select {
	case <-promise.ReadyChan():
		_, err := promise.Current()
		if err == nil {
			t.Fatal("expected error from promise, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("promise did not resolve within timeout")
	}

	select {
	case <-fatalChan:
		// Error was reported to fatalErrChan — this is the contract
	default:
		t.Fatal("expected error on fatalErrChan, got none")
	}
}

func TestComparatorClient_Reorg_Match(t *testing.T) {
	results := []*execution.MessageResult{
		{BlockHash: common.HexToHash("0x1111")},
		{BlockHash: common.HexToHash("0x2222")},
	}
	internal := &fakeExecClient{reorgResults: results}
	external := &fakeExecClient{reorgResults: results}

	c := NewComparatorClient(internal, external, make(chan error, 1))
	promise := c.Reorg(0, nil)

	select {
	case <-promise.ReadyChan():
		res, err := promise.Current()
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(res) != 2 {
			t.Fatalf("expected 2 results, got %d", len(res))
		}
		if res[0].BlockHash != results[0].BlockHash {
			t.Fatalf("expected first hash %v, got %v", results[0].BlockHash, res[0].BlockHash)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("promise did not resolve within timeout")
	}
}

func TestComparatorClient_Start_BothSucceed(t *testing.T) {
	internal := &fakeExecClient{}
	external := &fakeExecClient{}

	c := NewComparatorClient(internal, external, nil)
	err := c.Start(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !internal.startCalled {
		t.Fatal("expected internal Start to be called")
	}
	if !external.startCalled {
		t.Fatal("expected external Start to be called")
	}
}

func TestComparatorClient_Start_ExternalFails(t *testing.T) {
	internal := &fakeExecClient{}
	external := &fakeExecClient{startErr: context.DeadlineExceeded}

	c := NewComparatorClient(internal, external, nil)
	err := c.Start(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded wrapped error, got: %v", err)
	}
}

func TestComparatorClient_Initialize_DelegatesToBothBridges(t *testing.T) {
	internal := &fakeBridgeClient{}
	external := &fakeBridgeClient{}

	c := NewComparatorClient(internal, external, nil)
	err := c.Initialize(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !internal.initializeCalled {
		t.Fatal("expected internal Initialize to be called")
	}
	if !external.initializeCalled {
		t.Fatal("expected external Initialize to be called")
	}
}

func TestComparatorClient_SetConsensusClient_DelegatesToBothBridges(t *testing.T) {
	internal := &fakeBridgeClient{}
	external := &fakeBridgeClient{}

	c := NewComparatorClient(internal, external, nil)
	c.SetConsensusClient(nil)
	if !internal.setConsensusCalled {
		t.Fatal("expected internal SetConsensusClient to be called")
	}
	if !external.setConsensusCalled {
		t.Fatal("expected external SetConsensusClient to be called")
	}
}
