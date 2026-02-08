package nethexec

import (
	"context"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

var (
	defaultUrl   = "http://localhost:20545"
	defaultWsUrl = "ws://localhost:28551"
)

type NethRpcClient struct {
	client *rpc.Client
	url    string
	wsUrl  string
}

type messageParams struct {
	Index              arbutil.MessageIndex            `json:"index"`
	Message            *arbostypes.MessageWithMetadata `json:"message"`
	MessageForPrefetch *arbostypes.MessageWithMetadata `json:"messageForPrefetch,omitempty"`
}

type initializeMessageParams struct {
	InitialL1BaseFee      *big.Int `json:"initialL1BaseFee"`
	SerializedChainConfig []byte   `json:"serializedChainConfig"`
}

type setFinalityDataParams struct {
	SafeFinalityData      *rpcFinalityData `json:"safeFinalityData,omitempty"`
	FinalizedFinalityData *rpcFinalityData `json:"finalizedFinalityData,omitempty"`
	ValidatedFinalityData *rpcFinalityData `json:"validatedFinalityData,omitempty"`
}

type rpcFinalityData struct {
	MsgIdx    uint64      `json:"msgIdx"`
	BlockHash common.Hash `json:"blockHash"`
}

type reorgParams struct {
	Index   arbutil.MessageIndex                         `json:"index"`
	Message []arbostypes.MessageWithMetadataAndBlockInfo `json:"message"`
}

type setConsensusSyncDataParams struct {
	Synced          bool                   `json:"synced"`
	MaxMessageCount uint64                 `json:"maxMessageCount"`
	SyncProgressMap map[string]interface{} `json:"syncProgressMap,omitempty"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

// RPC intermediate types for deserializing Nethermind sequencer responses.
// Field names must match C# [JsonPropertyName] attributes.
type rpcMsgResult struct {
	Hash     common.Hash `json:"hash"`
	SendRoot common.Hash `json:"sendRoot"`
}

type rpcSequencedMsg struct {
	MsgIdx        uint64                         `json:"msgIdx"`
	MsgWithMeta   arbostypes.MessageWithMetadata `json:"msgWithMeta"`
	MsgResult     *rpcMsgResult                  `json:"msgResult"`
	BlockMetadata []byte                         `json:"blockMetadata"`
}

func (r *rpcSequencedMsg) toSequencedMsg() *execution.SequencedMsg {
	if r == nil {
		return nil
	}
	msg := &execution.SequencedMsg{
		MsgIdx:        arbutil.MessageIndex(r.MsgIdx),
		MsgWithMeta:   r.MsgWithMeta,
		BlockMetadata: common.BlockMetadata(r.BlockMetadata),
	}
	if r.MsgResult != nil {
		msg.MsgResult = &execution.MessageResult{
			BlockHash: r.MsgResult.Hash,
			SendRoot:  r.MsgResult.SendRoot,
		}
	}
	return msg
}

type rpcStartSequencingResult struct {
	SequencedMsg   *rpcSequencedMsg `json:"sequencedMsg"`
	WaitDurationMs int64            `json:"waitDurationMs"`
}

// InitMessageDigester is an interface for processing init messages
type InitMessageDigester interface {
	DigestInitMessage(ctx context.Context, initialL1BaseFee *big.Int, serializedChainConfig []byte) *execution.MessageResult
}

type fakeRemoteExecutionRpcClient struct{}

func NewFakeRemoteExecutionRpcClient() *fakeRemoteExecutionRpcClient {
	return &fakeRemoteExecutionRpcClient{}
}

func (n *fakeRemoteExecutionRpcClient) DigestInitMessage(context.Context, *big.Int, []byte) *execution.MessageResult {
	return &execution.MessageResult{}
}

var (
	_ InitMessageDigester = (*fakeRemoteExecutionRpcClient)(nil)
	_ InitMessageDigester = (*NethRpcClient)(nil)
)

func NewNethRpcClient(url string, wsUrl string) (*NethRpcClient, error) {
	if url == "" {
		log.Warn("No Nethermind URL provided, using default", "url", defaultUrl)
		url = defaultUrl
	}

	httpClient := rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	})

	// WebSocket is optional - only needed for subscriptions
	if wsUrl == "" {
		log.Info("No Nethermind WebSocket URL provided, subscriptions will not be available")
		wsUrl = defaultWsUrl // Fallback to default
	}

	ctx := context.Background()
	rpcClient, err := rpc.DialOptions(ctx, url, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create Neth RPC client: %w", err)
	}

	log.Info("Created Neth RPC client", "url", url, "wsUrl", wsUrl)

	return &NethRpcClient{
		client: rpcClient,
		url:    url,
		wsUrl:  wsUrl,
	}, nil
}

func (c *NethRpcClient) Close() {
	c.client.Close()
}

func (c *NethRpcClient) GetWebSocketURL() string {
	return c.wsUrl
}

func (c *NethRpcClient) DigestMessage(ctx context.Context, index arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) *execution.MessageResult {
	params := messageParams{
		Index:              index,
		Message:            msg,
		MessageForPrefetch: msgForPrefetch,
	}

	log.Debug("Making JSON-RPC call to DigestMessage",
		"url", c.url,
		"index", index,
		"messageType", msg.Message.Header.Kind,
	)

	var result execution.MessageResult
	if err := c.client.CallContext(ctx, &result, "DigestMessage", params); err != nil {
		log.Error("Failed to call DigestMessage", "error", err)
		return nil
	}

	return &result
}

func (c *NethRpcClient) DigestInitMessage(ctx context.Context, initialL1BaseFee *big.Int, serializedChainConfig []byte) *execution.MessageResult {
	var result execution.MessageResult

	params := initializeMessageParams{
		InitialL1BaseFee:      initialL1BaseFee,
		SerializedChainConfig: serializedChainConfig,
	}

	log.Debug("Making JSON-RPC call to DigestInitMessage",
		"url", c.url,
		"initialL1BaseFee", initialL1BaseFee,
		"len(serializedChainConfig)", len(serializedChainConfig))

	if err := c.client.CallContext(ctx, &result, "DigestInitMessage", params); err != nil {
		panic(fmt.Sprintf("failed to call DigestInitMessage: %v", err))
	}

	return &result
}

func (c *NethRpcClient) SetFinalityData(ctx context.Context, safeFinalityData *arbutil.FinalityData, finalizedFinalityData *arbutil.FinalityData, validatedFinalityData *arbutil.FinalityData) error {
	params := setFinalityDataParams{
		SafeFinalityData:      convertToRpcFinalityData(safeFinalityData),
		FinalizedFinalityData: convertToRpcFinalityData(finalizedFinalityData),
		ValidatedFinalityData: convertToRpcFinalityData(validatedFinalityData),
	}

	log.Debug("Making JSON-RPC call to SetFinalityData",
		"url", c.url,
		"safeFinalityData", safeFinalityData,
		"finalizedFinalityData", finalizedFinalityData,
		"validatedFinalityData", validatedFinalityData)

	var result any
	if err := c.client.CallContext(ctx, &result, "SetFinalityData", params); err != nil {
		log.Error("Failed to call SetFinalityData", "error", err)
		return fmt.Errorf("failed to call SetFinalityData: %w", err)
	}

	return nil
}

func convertToRpcFinalityData(data *arbutil.FinalityData) *rpcFinalityData {
	if data == nil {
		return nil
	}
	return &rpcFinalityData{
		MsgIdx:    uint64(data.MsgIdx),
		BlockHash: data.BlockHash,
	}
}

func (c *NethRpcClient) HeadMessageIndex(ctx context.Context) (arbutil.MessageIndex, error) {
	var result hexutil.Uint64
	if err := c.client.CallContext(ctx, &result, "HeadMessageIndex"); err != nil {
		log.Error("Failed to call HeadMessageIndex", "error", err)
		return 0, fmt.Errorf("failed to call HeadMessageIndex: %w", err)
	}
	return arbutil.MessageIndex(result), nil
}

func (c *NethRpcClient) ResultAtMessageIndex(ctx context.Context, index arbutil.MessageIndex) (*execution.MessageResult, error) {
	log.Debug("Making JSON-RPC call to ResultAtMessageIndex", "url", c.url, "index", index)
	var result execution.MessageResult
	if err := c.client.CallContext(ctx, &result, "ResultAtMessageIndex", uint64(index)); err != nil {
		log.Error("Failed to call ResultAtMessageIndex", "error", err)
		return nil, fmt.Errorf("failed to call ResultAtMessageIndex: %w", err)
	}
	return &result, nil
}

func (c *NethRpcClient) MessageIndexToBlockNumber(ctx context.Context, messageIndex arbutil.MessageIndex) (uint64, error) {
	log.Debug("Making JSON-RPC call to MessageIndexToBlockNumber", "url", c.url, "messageIndex", messageIndex)
	var result hexutil.Uint64
	if err := c.client.CallContext(ctx, &result, "MessageIndexToBlockNumber", uint64(messageIndex)); err != nil {
		log.Error("Failed to call MessageIndexToBlockNumber", "error", err)
		return 0, fmt.Errorf("failed to call MessageIndexToBlockNumber: %w", err)
	}
	return uint64(result), nil
}

func (c *NethRpcClient) BlockNumberToMessageIndex(ctx context.Context, blockNum uint64) (arbutil.MessageIndex, error) {
	log.Debug("Making JSON-RPC call to BlockNumberToMessageIndex", "url", c.url, "blockNum", blockNum)
	var result hexutil.Uint64
	if err := c.client.CallContext(ctx, &result, "BlockNumberToMessageIndex", blockNum); err != nil {
		log.Error("Failed to call BlockNumberToMessageIndex", "error", err)
		return 0, fmt.Errorf("failed to call BlockNumberToMessageIndex: %w", err)
	}
	return arbutil.MessageIndex(result), nil
}

func (c *NethRpcClient) MarkFeedStart(ctx context.Context, to arbutil.MessageIndex) error {
	var result string
	if err := c.client.CallContext(ctx, &result, "MarkFeedStart", uint64(to)); err != nil {
		log.Error("Failed to call MarkFeedStart", "error", err, "to", to)
		return fmt.Errorf("failed to call MarkFeedStart: %w", err)
	}
	return nil
}

// Reorg handles chain reorganizations. Note: main Nitro interface doesn't include oldMessages parameter.
func (c *NethRpcClient) Reorg(ctx context.Context, count arbutil.MessageIndex, newMessages []arbostypes.MessageWithMetadataAndBlockInfo) ([]*execution.MessageResult, error) {
	log.Debug("Making JSON-RPC call to Reorg", "url", c.url, "count", count, "newCount", len(newMessages))
	params := reorgParams{Index: count, Message: newMessages}
	var result []*execution.MessageResult
	if err := c.client.CallContext(ctx, &result, "Reorg", params); err != nil {
		log.Error("Failed to call Reorg", "error", err)
		return nil, fmt.Errorf("failed to call Reorg: %w", err)
	}
	return result, nil
}

func (c *NethRpcClient) SetConsensusSyncData(ctx context.Context, syncData *execution.ConsensusSyncData) error {
	if syncData == nil {
		return fmt.Errorf("syncData cannot be nil")
	}

	params := setConsensusSyncDataParams{
		Synced:          syncData.Synced,
		MaxMessageCount: uint64(syncData.MaxMessageCount),
		SyncProgressMap: syncData.SyncProgressMap,
		UpdatedAt:       syncData.UpdatedAt,
	}

	log.Debug("Making JSON-RPC call to SetConsensusSyncData",
		"url", c.url,
		"synced", syncData.Synced,
		"maxMessageCount", syncData.MaxMessageCount,
		"updatedAt", syncData.UpdatedAt)

	var result any
	if err := c.client.CallContext(ctx, &result, "SetConsensusSyncData", params); err != nil {
		log.Error("Failed to call SetConsensusSyncData", "error", err)
		return fmt.Errorf("failed to call SetConsensusSyncData: %w", err)
	}

	return nil
}

func (c *NethRpcClient) Synced(ctx context.Context) (bool, error) {
	log.Debug("Making JSON-RPC call to Synced", "url", c.url)
	var result bool
	if err := c.client.CallContext(ctx, &result, "Synced"); err != nil {
		log.Error("Failed to call Synced", "error", err)
		return false, fmt.Errorf("failed to call Synced: %w", err)
	}
	return result, nil
}

func (c *NethRpcClient) FullSyncProgressMap(ctx context.Context) (map[string]interface{}, error) {
	log.Debug("Making JSON-RPC call to FullSyncProgressMap", "url", c.url)
	var result map[string]interface{}
	if err := c.client.CallContext(ctx, &result, "FullSyncProgressMap"); err != nil {
		log.Error("Failed to call FullSyncProgressMap", "error", err)
		return nil, fmt.Errorf("failed to call FullSyncProgressMap: %w", err)
	}
	return result, nil
}

func (c *NethRpcClient) StartSequencing(ctx context.Context) (*execution.SequencedMsg, time.Duration, error) {
	log.Debug("Making JSON-RPC call to nitroexecution_startSequencing", "url", c.url)
	var result rpcStartSequencingResult
	if err := c.client.CallContext(ctx, &result, "nitroexecution_startSequencing"); err != nil {
		log.Error("Failed to call nitroexecution_startSequencing", "error", err)
		return nil, 0, fmt.Errorf("failed to call nitroexecution_startSequencing: %w", err)
	}
	return result.SequencedMsg.toSequencedMsg(), time.Duration(result.WaitDurationMs) * time.Millisecond, nil
}

func (c *NethRpcClient) EndSequencing(ctx context.Context, errWhileSequencing error) error {
	log.Debug("Making JSON-RPC call to nitroexecution_endSequencing", "url", c.url)
	var errStr *string
	if errWhileSequencing != nil {
		s := errWhileSequencing.Error()
		errStr = &s
	}
	var result any
	if err := c.client.CallContext(ctx, &result, "nitroexecution_endSequencing", errStr); err != nil {
		log.Error("Failed to call nitroexecution_endSequencing", "error", err)
		return fmt.Errorf("failed to call nitroexecution_endSequencing: %w", err)
	}
	return nil
}

func (c *NethRpcClient) EnqueueDelayedMessages(ctx context.Context, msgs []*arbostypes.L1IncomingMessage, firstMsgIdx uint64) error {
	log.Debug("Making JSON-RPC call to nitroexecution_enqueueDelayedMessages",
		"url", c.url, "count", len(msgs), "firstMsgIdx", firstMsgIdx)
	var result any
	if err := c.client.CallContext(ctx, &result, "nitroexecution_enqueueDelayedMessages", msgs, firstMsgIdx); err != nil {
		log.Error("Failed to call nitroexecution_enqueueDelayedMessages", "error", err)
		return fmt.Errorf("failed to call nitroexecution_enqueueDelayedMessages: %w", err)
	}
	return nil
}

func (c *NethRpcClient) AppendLastSequencedBlock(ctx context.Context) error {
	log.Debug("Making JSON-RPC call to nitroexecution_appendLastSequencedBlock", "url", c.url)
	var result any
	if err := c.client.CallContext(ctx, &result, "nitroexecution_appendLastSequencedBlock"); err != nil {
		log.Error("Failed to call nitroexecution_appendLastSequencedBlock", "error", err)
		return fmt.Errorf("failed to call nitroexecution_appendLastSequencedBlock: %w", err)
	}
	return nil
}

func (c *NethRpcClient) NextDelayedMessageNumber(ctx context.Context) (uint64, error) {
	log.Debug("Making JSON-RPC call to nitroexecution_nextDelayedMessageNumber", "url", c.url)
	var result hexutil.Uint64
	if err := c.client.CallContext(ctx, &result, "nitroexecution_nextDelayedMessageNumber"); err != nil {
		log.Error("Failed to call nitroexecution_nextDelayedMessageNumber", "error", err)
		return 0, fmt.Errorf("failed to call nitroexecution_nextDelayedMessageNumber: %w", err)
	}
	return uint64(result), nil
}

func (c *NethRpcClient) ResequenceReorgedMessage(ctx context.Context, msg *arbostypes.MessageWithMetadata) (*execution.SequencedMsg, error) {
	log.Debug("Making JSON-RPC call to nitroexecution_resequenceReorgedMessage", "url", c.url)
	var result *rpcSequencedMsg
	if err := c.client.CallContext(ctx, &result, "nitroexecution_resequenceReorgedMessage", msg); err != nil {
		log.Error("Failed to call nitroexecution_resequenceReorgedMessage", "error", err)
		return nil, fmt.Errorf("failed to call nitroexecution_resequenceReorgedMessage: %w", err)
	}
	return result.toSequencedMsg(), nil
}

func (c *NethRpcClient) Pause(ctx context.Context) error {
	log.Debug("Making JSON-RPC call to nitroexecution_pause", "url", c.url)
	var result any
	if err := c.client.CallContext(ctx, &result, "nitroexecution_pause"); err != nil {
		log.Error("Failed to call nitroexecution_pause", "error", err)
		return fmt.Errorf("failed to call nitroexecution_pause: %w", err)
	}
	return nil
}

func (c *NethRpcClient) Activate(ctx context.Context) error {
	log.Debug("Making JSON-RPC call to nitroexecution_activate", "url", c.url)
	var result any
	if err := c.client.CallContext(ctx, &result, "nitroexecution_activate"); err != nil {
		log.Error("Failed to call nitroexecution_activate", "error", err)
		return fmt.Errorf("failed to call nitroexecution_activate: %w", err)
	}
	return nil
}

func (c *NethRpcClient) ForwardTo(ctx context.Context, url string) error {
	log.Debug("Making JSON-RPC call to nitroexecution_forwardTo", "url", c.url, "forwardUrl", url)
	var result any
	if err := c.client.CallContext(ctx, &result, "nitroexecution_forwardTo", url); err != nil {
		log.Error("Failed to call nitroexecution_forwardTo", "error", err)
		return fmt.Errorf("failed to call nitroexecution_forwardTo: %w", err)
	}
	return nil
}
