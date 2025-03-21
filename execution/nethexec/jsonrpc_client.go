package nethexec

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
	"net/http"
	"time"
)

// JSONRPCClient is a client for making JSON-RPC calls to the Arbitrum node
type JSONRPCClient struct {
	client *rpc.Client
	url    string
}

// NewJSONRPCClient creates a new client for making JSON-RPC calls
func NewJSONRPCClient(url string) (*JSONRPCClient, error) {
	httpClient := rpc.WithHTTPClient(&http.Client{
		Timeout: 30 * time.Second,
	})

	context := context.Background()
	rpcClient, err := rpc.DialOptions(context, url, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}

	return &JSONRPCClient{
		client: rpcClient,
		url:    url,
	}, nil
}

// Close closes the RPC client
func (c *JSONRPCClient) Close() {
	c.client.Close()
}

// MessageParams represents the parameters for the arbitrum_digestMessage RPC call
type MessageParams struct {
	Number             arbutil.MessageIndex            `json:"number"`
	Message            *arbostypes.MessageWithMetadata `json:"message"`
	MessageForPrefetch *arbostypes.MessageWithMetadata `json:"messageForPrefetch,omitempty"`
}

// digestMessageAsync makes an async JSON-RPC call to arbitrum_digestMessage
func (c *JSONRPCClient) digestMessageAsync(ctx context.Context, params MessageParams) containers.PromiseInterface[*execution.MessageResult] {
	promise := containers.NewPromise[*execution.MessageResult](nil)

	go func() {
		var result execution.MessageResult
		err := c.client.CallContext(ctx, &result, "arbitrum_digestMessage", params)
		if err != nil {
			promise.ProduceError(err)
			return
		}

		promise.Produce(&result)
	}()

	return &promise
}

// DigestMessage makes a JSON-RPC call to arbitrum_digestMessage
func (c *JSONRPCClient) DigestMessage(ctx context.Context, num arbutil.MessageIndex, msg *arbostypes.MessageWithMetadata, msgForPrefetch *arbostypes.MessageWithMetadata) containers.PromiseInterface[*execution.MessageResult] {
	params := MessageParams{
		Number:             num,
		Message:            msg,
		MessageForPrefetch: msgForPrefetch,
	}

	log.Info("Making JSON-RPC call to arbitrum_digestMessage",
		"url", c.url,
		"num", num,
		"messageType", msg.Message.Header.Kind)

	return c.digestMessageAsync(ctx, params)
}
