// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package consensusrpcserver

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution"
)

type ConsensusRPCServer struct {
	info execution.ConsensusInfo
}

func NewConsensusRPCServer(info execution.ConsensusInfo) *ConsensusRPCServer {
	return &ConsensusRPCServer{info}
}

func (s *ConsensusRPCServer) BlockMetadataAtMessageIndex(ctx context.Context, msgIdx arbutil.MessageIndex) (common.BlockMetadata, error) {
	return s.info.BlockMetadataAtMessageIndex(msgIdx).Await(ctx)
}
