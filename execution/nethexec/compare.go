package nethexec

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	"github.com/offchainlabs/nitro/util/containers"
)

// comparePromises awaits two promises concurrently, compares their results,
// records duration metrics, and handles divergence reporting.
// The internal (Geth) result is always returned as canonical.
// If fatalErrChan is non-nil, mismatches are sent to it and the promise errors.
// If fatalErrChan is nil, mismatches are logged and the internal result is returned.
func comparePromises[T any](fatalErrChan chan error, op string,
	internal containers.PromiseInterface[T],
	external containers.PromiseInterface[T],
) containers.PromiseInterface[T] {
	promise := containers.NewPromise[T](nil)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var intRes, extRes T
		var intErr, extErr error
		var internalDuration, externalDuration time.Duration

		internalStart := time.Now()
		externalStart := time.Now()

		internalDone := false
		externalDone := false

		for !internalDone || !externalDone {
			select {
			case <-internal.ReadyChan():
				if !internalDone {
					intRes, intErr = internal.Current()
					internalDuration = time.Since(internalStart)
					internalDone = true
				}
			case <-external.ReadyChan():
				if !externalDone {
					extRes, extErr = external.Current()
					externalDuration = time.Since(externalStart)
					externalDone = true
				}
			case <-ctx.Done():
				promise.ProduceError(ctx.Err())
				return
			}
		}

		metrics.GetOrRegisterHistogram("arb/compare/internal/"+op+"/duration", nil, metrics.NewBoundedHistogramSample()).Update(internalDuration.Nanoseconds())
		metrics.GetOrRegisterHistogram("arb/compare/external/"+op+"/duration", nil, metrics.NewBoundedHistogramSample()).Update(externalDuration.Nanoseconds())

		if err := compare(op, intRes, intErr, extRes, extErr); err != nil {
			if fatalErrChan != nil {
				reportDivergence(fatalErrChan, op, err)
				promise.ProduceError(err)
			} else {
				log.Error("Non-fatal comparison error", "operation", op, "err", err)
				promise.ProduceResultSafe(intRes, intErr)
			}
		} else {
			promise.ProduceResultSafe(intRes, intErr)
		}
	}()
	return &promise
}

// reportDivergence sends a comparison error to fatalErrChan (non-blocking).
// Used by both comparePromises and ComparatorSequencer for consistent divergence handling.
func reportDivergence(fatalErrChan chan error, op string, err error) {
	select {
	case fatalErrChan <- fmt.Errorf("compareExecutionClient %s: %s", op, err.Error()):
	default:
		log.Warn("fatalErrChan full, dropping comparison error", "op", op, "err", err)
	}
}

// compare performs deep-equality comparison of two operation results.
// It handles all four error combinations and uses go-cmp for value diffing.
func compare[T any](op string, intRes T, intErr error, extRes T, extErr error) error {
	switch {
	case intErr != nil && extErr != nil:
		return fmt.Errorf("both operations failed: internal=%w external=%s", intErr, extErr.Error())
	case intErr != nil && extErr == nil:
		return fmt.Errorf("internal operation failed: %w", intErr)
	case intErr == nil && extErr != nil:
		return fmt.Errorf("external operation failed: %w", extErr)
	default:
		opts := cmp.Options{
			cmp.Transformer("HashHex", func(h common.Hash) string { return h.Hex() }),
			cmp.Comparer(func(a, b *big.Int) bool {
				if a == nil && b == nil {
					return true
				}
				if a == nil || b == nil {
					return false
				}
				return a.Cmp(b) == 0
			}),
			cmpopts.EquateEmpty(),
		}
		if diff := cmp.Diff(intRes, extRes, opts...); diff != "" {
			fmt.Printf("ERROR: Execution mismatch detected in operation: %s\n", op)
			fmt.Printf("Diff details:\n%s\n", diff)
			return fmt.Errorf("execution mismatch in %s", op)
		}
	}
	return nil
}
