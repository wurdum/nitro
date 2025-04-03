package nethexec

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/containers"
)

type NodeWrapper struct {
	*gethexec.ExecutionNode

	rpcClient     *JSONRPCClient
	rpcClientLock sync.Mutex
	rpcURL        string
}

func NewNodeWrapper(node *gethexec.ExecutionNode) *NodeWrapper {
	return &NodeWrapper{
		ExecutionNode: node,
		rpcURL:        "http://localhost:7547",
	}
}

func (w *NodeWrapper) SetRPCURL(url string) {
	w.rpcClientLock.Lock()
	defer w.rpcClientLock.Unlock()

	if w.rpcClient != nil {
		w.rpcClient.Close()
		w.rpcClient = nil
	}

	w.rpcURL = url
}

func (w *NodeWrapper) getRPCClient() (*JSONRPCClient, error) {
	w.rpcClientLock.Lock()
	defer w.rpcClientLock.Unlock()

	if w.rpcClient == nil {
		client, err := NewJSONRPCClient(w.rpcURL)
		if err != nil {
			return nil, err
		}
		w.rpcClient = client
	}

	return w.rpcClient, nil
}

// ---- execution.ExecutionClient interface methods ----

func (w *NodeWrapper) DigestMessage(num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	start := time.Now()
	log.Info("NodeWrapper: DigestMessage", "num", num)

	//client, err := w.getRPCClient()
	//if err == nil {
	//	result := client.DigestMessage(context.Background(), num, msg, msgForPrefetch)
	//	log.Info("NodeWrapper: DigestMessage via JSON-RPC completed", "num", num, "elapsed", time.Since(start))
	//	return result
	//}
	//
	//log.Warn("Failed to get RPC client, falling back to direct call", "err", err)
	result := w.ExecutionNode.DigestMessage(num, msg, msgForPrefetch)
	log.Info("NodeWrapper: DigestMessage via direct call completed", "num", num, "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) Reorg(count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo, oldMessages []*arbostypes.MessageWithMetadata) containers.PromiseInterface[[]*execution.MessageResult] {
	start := time.Now()
	log.Info("NodeWrapper: Reorg", "count", count, "newMessagesCount", len(newMessages), "oldMessagesCount", len(oldMessages))
	result := w.ExecutionNode.Reorg(count, newMessages, oldMessages)
	log.Info("NodeWrapper: Reorg completed", "count", count, "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) HeadMessageIndex() containers.PromiseInterface[arbutil.MessageIndex] {
	//start := time.Now()
	//log.Info("NodeWrapper: HeadMessageIndex")
	result := w.ExecutionNode.HeadMessageIndex()
	//log.Info("NodeWrapper: HeadMessageIndex completed", "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) ResultAtMessageIndex(pos arbutil.MessageIndex) containers.PromiseInterface[*execution.MessageResult] {
	start := time.Now()
	log.Info("NodeWrapper: ResultAtMessageIndex", "pos", pos)
	result := w.ExecutionNode.ResultAtMessageIndex(pos)
	log.Info("NodeWrapper: ResultAtMessageIndex completed", "pos", pos, "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) MessageIndexToBlockNumber(messageNum arbutil.MessageIndex) containers.PromiseInterface[uint64] {
	start := time.Now()
	log.Info("NodeWrapper: MessageIndexToBlockNumber", "messageNum", messageNum)
	result := w.ExecutionNode.MessageIndexToBlockNumber(messageNum)
	log.Info("NodeWrapper: MessageIndexToBlockNumber completed", "messageNum", messageNum, "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) BlockNumberToMessageIndex(blockNum uint64) containers.PromiseInterface[arbutil.MessageIndex] {
	start := time.Now()
	log.Info("NodeWrapper: BlockNumberToMessageIndex", "blockNum", blockNum)
	result := w.ExecutionNode.BlockNumberToMessageIndex(blockNum)
	log.Info("NodeWrapper: BlockNumberToMessageIndex completed", "blockNum", blockNum, "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) SetFinalityData(ctx context.Context, finalityData *arbutil.FinalityData) containers.PromiseInterface[struct{}] {
	//start := time.Now()
	//log.Info("NodeWrapper: SetFinalityData")
	result := w.ExecutionNode.SetFinalityData(ctx, finalityData)
	//log.Info("NodeWrapper: SetFinalityData completed", "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) MarkFeedStart(to arbutil.MessageIndex) containers.PromiseInterface[struct{}] {
	start := time.Now()
	log.Info("NodeWrapper: MarkFeedStart", "to", to)
	result := w.ExecutionNode.MarkFeedStart(to)
	log.Info("NodeWrapper: MarkFeedStart completed", "to", to, "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) Maintenance() containers.PromiseInterface[struct{}] {
	start := time.Now()
	log.Info("NodeWrapper: Maintenance")
	result := w.ExecutionNode.Maintenance()
	log.Info("NodeWrapper: Maintenance completed", "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) Start(ctx context.Context) containers.PromiseInterface[struct{}] {
	start := time.Now()
	log.Info("NodeWrapper: Start")
	result := w.ExecutionNode.Start(ctx)
	log.Info("NodeWrapper: Start completed", "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) StopAndWait() containers.PromiseInterface[struct{}] {
	start := time.Now()
	log.Info("NodeWrapper: StopAndWait")
	result := w.ExecutionNode.StopAndWait()
	log.Info("NodeWrapper: StopAndWait completed", "elapsed", time.Since(start))
	return result
}

// ---- execution.ExecutionSequencer interface methods ----

func (w *NodeWrapper) Pause() {
	start := time.Now()
	log.Info("NodeWrapper: Pause")
	w.ExecutionNode.Pause()
	log.Info("NodeWrapper: Pause completed", "elapsed", time.Since(start))
}

func (w *NodeWrapper) Activate() {
	start := time.Now()
	log.Info("NodeWrapper: Activate")
	w.ExecutionNode.Activate()
	log.Info("NodeWrapper: Activate completed", "elapsed", time.Since(start))
}

func (w *NodeWrapper) ForwardTo(url string) error {
	start := time.Now()
	log.Info("NodeWrapper: ForwardTo", "url", url)
	err := w.ExecutionNode.ForwardTo(url)
	log.Info("NodeWrapper: ForwardTo completed", "url", url, "err", err, "elapsed", time.Since(start))
	return err
}

func (w *NodeWrapper) SequenceDelayedMessage(message *arbostypes.L1IncomingMessage, delayedSeqNum uint64) error {
	start := time.Now()
	log.Info("NodeWrapper: SequenceDelayedMessage", "delayedSeqNum", delayedSeqNum)
	err := w.ExecutionNode.SequenceDelayedMessage(message, delayedSeqNum)
	log.Info("NodeWrapper: SequenceDelayedMessage completed", "delayedSeqNum", delayedSeqNum, "err", err, "elapsed", time.Since(start))
	return err
}

func (w *NodeWrapper) NextDelayedMessageNumber() (uint64, error) {
	//start := time.Now()
	//log.Info("NodeWrapper: NextDelayedMessageNumber")
	result, err := w.ExecutionNode.NextDelayedMessageNumber()
	//log.Info("NodeWrapper: NextDelayedMessageNumber completed", "result", result, "err", err, "elapsed", time.Since(start))
	return result, err
}

func (w *NodeWrapper) Synced(ctx context.Context) bool {
	start := time.Now()
	log.Info("NodeWrapper: Synced")
	result := w.ExecutionNode.Synced(ctx)
	log.Info("NodeWrapper: Synced completed", "result", result, "elapsed", time.Since(start))
	return result
}

func (w *NodeWrapper) FullSyncProgressMap(ctx context.Context) map[string]interface{} {
	start := time.Now()
	log.Info("NodeWrapper: FullSyncProgressMap")
	result := w.ExecutionNode.FullSyncProgressMap(ctx)
	log.Info("NodeWrapper: FullSyncProgressMap completed", "elapsed", time.Since(start))
	return result
}

// ---- execution.ExecutionRecorder interface methods ----

func (w *NodeWrapper) RecordBlockCreation(ctx context.Context, pos arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata) (*execution.RecordResult, error) {
	start := time.Now()
	log.Info("NodeWrapper: RecordBlockCreation", "pos", pos)
	result, err := w.ExecutionNode.RecordBlockCreation(ctx, pos, msg)
	log.Info("NodeWrapper: RecordBlockCreation completed", "pos", pos, "err", err, "elapsed", time.Since(start))
	return result, err
}

func (w *NodeWrapper) MarkValid(pos arbutil.MessageIndex, resultHash common.Hash) {
	start := time.Now()
	log.Info("NodeWrapper: MarkValid", "pos", pos, "resultHash", resultHash)
	w.ExecutionNode.MarkValid(pos, resultHash)
	log.Info("NodeWrapper: MarkValid completed", "pos", pos, "elapsed", time.Since(start))
}

func (w *NodeWrapper) PrepareForRecord(ctx context.Context, start, end arbutil.MessageIndex) error {
	startTime := time.Now()
	log.Info("NodeWrapper: PrepareForRecord", "start", start, "end", end)
	err := w.ExecutionNode.PrepareForRecord(ctx, start, end)
	log.Info("NodeWrapper: PrepareForRecord completed", "start", start, "end", end, "err", err, "elapsed", time.Since(startTime))
	return err
}

// ---- execution.ExecutionBatchPoster interface methods ----

func (w *NodeWrapper) ArbOSVersionForMessageIndex(msgIdx arbutil.MessageIndex) (uint64, error) {
	start := time.Now()
	log.Info("NodeWrapper: ArbOSVersionForMessageIndex", "msgIdx", msgIdx)
	result, err := w.ExecutionNode.ArbOSVersionForMessageIndex(msgIdx)
	log.Info("NodeWrapper: ArbOSVersionForMessageIndex completed", "msgIdx", msgIdx, "result", result, "err", err, "elapsed", time.Since(start))
	return result, err
}
