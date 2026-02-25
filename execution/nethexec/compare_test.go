package nethexec

import (
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/offchainlabs/nitro/execution"
	"github.com/offchainlabs/nitro/util/containers"
)

func TestCompare_BothSucceed_Equal(t *testing.T) {
	result := &execution.MessageResult{
		BlockHash: common.HexToHash("0x1234"),
		SendRoot:  common.HexToHash("0x5678"),
	}
	err := compare("test", result, nil, result, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestCompare_BothSucceed_Different(t *testing.T) {
	intRes := &execution.MessageResult{
		BlockHash: common.HexToHash("0x1234"),
		SendRoot:  common.HexToHash("0x5678"),
	}
	extRes := &execution.MessageResult{
		BlockHash: common.HexToHash("0xaaaa"),
		SendRoot:  common.HexToHash("0x5678"),
	}
	err := compare("test", intRes, nil, extRes, nil)
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "execution mismatch") {
		t.Fatalf("expected mismatch error, got: %v", err)
	}
}

func TestCompare_InternalFails(t *testing.T) {
	intErr := errors.New("internal boom")
	extRes := &execution.MessageResult{
		BlockHash: common.HexToHash("0x1234"),
	}
	err := compare("test", (*execution.MessageResult)(nil), intErr, extRes, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "internal operation failed") {
		t.Fatalf("expected internal failure error, got: %v", err)
	}
}

func TestCompare_ExternalFails(t *testing.T) {
	intRes := &execution.MessageResult{
		BlockHash: common.HexToHash("0x1234"),
	}
	extErr := errors.New("external boom")
	err := compare("test", intRes, nil, (*execution.MessageResult)(nil), extErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "external operation failed") {
		t.Fatalf("expected external failure error, got: %v", err)
	}
}

func TestCompare_BothFail(t *testing.T) {
	intErr := errors.New("internal boom")
	extErr := errors.New("external boom")
	err := compare("test", (*execution.MessageResult)(nil), intErr, (*execution.MessageResult)(nil), extErr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "both operations failed") {
		t.Fatalf("expected both-failed error, got: %v", err)
	}
}

func TestCompare_BigIntEquality(t *testing.T) {
	type valHolder struct {
		Val *big.Int
	}
	a := valHolder{Val: big.NewInt(42)}
	b := valHolder{Val: big.NewInt(42)}
	err := compare("bigint-test", a, nil, b, nil)
	if err != nil {
		t.Fatalf("expected no error for equal big.Int values, got: %v", err)
	}
}

func TestCompare_BigIntInequality(t *testing.T) {
	type valHolder struct {
		Val *big.Int
	}
	a := valHolder{Val: big.NewInt(42)}
	b := valHolder{Val: big.NewInt(99)}
	err := compare("bigint-test", a, nil, b, nil)
	if err == nil {
		t.Fatal("expected mismatch error for different big.Int values, got nil")
	}
}

func TestComparePromises_BothSucceed_Match(t *testing.T) {
	result := &execution.MessageResult{
		BlockHash: common.HexToHash("0x1234"),
		SendRoot:  common.HexToHash("0x5678"),
	}
	internal := containers.NewReadyPromise(result, nil)
	external := containers.NewReadyPromise(result, nil)

	promise := comparePromises[*execution.MessageResult](nil, "test", internal, external)

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

func TestComparePromises_Mismatch_FatalChan(t *testing.T) {
	intRes := &execution.MessageResult{
		BlockHash: common.HexToHash("0x1234"),
	}
	extRes := &execution.MessageResult{
		BlockHash: common.HexToHash("0xaaaa"),
	}
	internal := containers.NewReadyPromise(intRes, nil)
	external := containers.NewReadyPromise(extRes, nil)

	fatalChan := make(chan error, 1)
	promise := comparePromises[*execution.MessageResult](fatalChan, "test", internal, external)

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
	case fatalErr := <-fatalChan:
		if !strings.Contains(fatalErr.Error(), "compareExecutionClient") {
			t.Fatalf("expected compareExecutionClient error, got: %v", fatalErr)
		}
	default:
		t.Fatal("expected error on fatalErrChan, got none")
	}
}

func TestComparePromises_Mismatch_NilChan(t *testing.T) {
	intRes := &execution.MessageResult{
		BlockHash: common.HexToHash("0x1234"),
	}
	extRes := &execution.MessageResult{
		BlockHash: common.HexToHash("0xaaaa"),
	}
	internal := containers.NewReadyPromise(intRes, nil)
	external := containers.NewReadyPromise(extRes, nil)

	promise := comparePromises[*execution.MessageResult](nil, "test", internal, external)

	select {
	case <-promise.ReadyChan():
		res, err := promise.Current()
		if err != nil {
			t.Fatalf("expected no error (non-fatal), got: %v", err)
		}
		if res.BlockHash != intRes.BlockHash {
			t.Fatalf("expected internal result returned, got %v", res.BlockHash)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("promise did not resolve within timeout")
	}
}
