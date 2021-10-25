//go:build gofuzzbeta
// +build gofuzzbeta

package network

import (
	"context"
	"testing"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func FuzzNetworkRPC(f *testing.F) {
	f.Fuzz(func(t *testing.T, msg []byte) {
		ethjson := new(ethtypes.Transaction)
		binerr := ethjson.UnmarshalBinary(msg)
		if binerr == nil {
			testnetwork := New(t, DefaultConfig())
			_, err := testnetwork.WaitForHeight(1)
			if err != nil {
				t.Log("failed to start up the network")
			} else if testnetwork.Validators != nil && len(testnetwork.Validators) > 0 && testnetwork.Validators[0].JSONRPCClient != nil {
				testnetwork.Validators[0].JSONRPCClient.SendTransaction(context.Background(), ethjson)
				h, err := testnetwork.WaitForHeightWithTimeout(10, time.Minute)
				if err != nil {
					testnetwork.Cleanup()
					t.Fatalf("expected to reach 10 blocks; got %d", h)
				}
				latestHeight, err := testnetwork.LatestHeight()
				if err != nil {
					testnetwork.Cleanup()
					t.Fatalf("latest height failed")
				}
				if latestHeight < h {
					testnetwork.Cleanup()
					t.Errorf("latestHeight should be greater or equal to")
				}
				testnetwork.Cleanup()
			}
		}
	})
}

func FuzzEVMHandler(f *testing.F) {

	f.Fuzz(func(t *testing.T, amount1 int64, gasLimit1 uint64, gasPrice1 int64, input1 []byte,
		amount2 int64, nonce2 uint64, gasLimit2 uint64, gasPrice2 int64, input2 []byte,
		amount3 int64, gasLimit3 uint64, gasPrice3 int64, input3 []byte) {

		err := FuzzEVMHandlerImpl(amount1, gasLimit1, gasPrice1, input1,
			amount2, nonce2, gasLimit2, gasPrice2, input2,
			amount3, gasLimit3, gasPrice3, input3)

		require.NoError(t, err)
	})
}
