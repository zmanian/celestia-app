package testnode

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/celestiaorg/celestia-app/testutil"
	"github.com/celestiaorg/celestia-app/x/payment"
	"github.com/celestiaorg/celestia-app/x/payment/types"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/pkg/consts"
)

// LatestHeight returns the latest height of the network or an error if the
// query fails.
func LatestHeight(cctx client.Context) (int64, error) {
	status, err := cctx.Client.Status(context.Background())
	if err != nil {
		return 0, err
	}

	return status.SyncInfo.LatestBlockHeight, nil
}

// WaitForHeightWithTimeout is the same as WaitForHeight except the caller can
// provide a custom timeout.
func WaitForHeightWithTimeout(cctx client.Context, h int64, t time.Duration) (int64, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(t)
	defer timeout.Stop()

	var latestHeight int64
	for {
		select {
		case <-timeout.C:
			return latestHeight, errors.New("timeout exceeded waiting for block")
		case <-ticker.C:
			latestHeight, err := LatestHeight(cctx)
			if err != nil {
				return 0, err
			}
			if latestHeight >= h {
				return latestHeight, nil
			}
		}
	}
}

// WaitForHeight performs a blocking check where it waits for a block to be
// committed after a given block. If that height is not reached within a timeout,
// an error is returned. Regardless, the latest height queried is returned.
func WaitForHeight(cctx client.Context, h int64) (int64, error) {
	return WaitForHeightWithTimeout(cctx, h, 10*time.Second)
}

// WaitForNextBlock waits for the next block to be committed, returning an error
// upon failure.
func WaitForNextBlock(cctx client.Context) error {
	lastBlock, err := LatestHeight(cctx)
	if err != nil {
		return err
	}

	_, err = WaitForHeight(cctx, lastBlock+1)
	if err != nil {
		return err
	}

	return err
}

// FillBlock will create and submit enough PFD txs to fill a block to a specific
// square size. It uses a crude mechanism to estimate the number of txs needed
// by creating message that each take up a single row, and creating squareSize
// -2 of those PFDs.
func FillBlock(cctx client.Context, squareSize int, accounts []string) ([]*sdk.TxResponse, error) {
	// todo: fix or debug this after cherry-picking this commit to a branch w/ non-interactive defaults
	msgCount := (squareSize / 4)
	if len(accounts) < msgCount {
		return nil, fmt.Errorf("more funded accounts are needed: want >=%d have %d", msgCount, len(accounts))
	}

	// todo: fix or debug this after cherry-picking this commit to a branch w/ non-interactive defaults
	msgSize := ((squareSize / 2) * consts.MsgShareSize) - 300

	opts := []types.TxBuilderOption{
		types.SetGasLimit(100000000000000),
	}

	results := make([]*sdk.TxResponse, msgCount)
	for i := 0; i < msgCount; i++ {
		// use the key for accounts[i] to create a singer used for a single PFD
		signer := types.NewKeyringSigner(cctx.Keyring, accounts[i], cctx.ChainID)

		// create a random msg per row
		pfd, err := payment.BuildPayForData(
			context.TODO(),
			signer,
			cctx.GRPCClient,
			testutil.RandomValidNamespace(),
			tmrand.Bytes(msgSize),
			opts...,
		)
		if err != nil {
			return nil, err
		}

		signed, err := payment.SignPayForData(signer, pfd, opts...)
		if err != nil {
			return nil, err
		}

		rawTx, err := signer.EncodeTx(signed)
		if err != nil {
			return nil, err
		}

		res, err := cctx.BroadcastTxSync(rawTx)
		if err != nil {
			return nil, err
		}
		if res.Code != abci.CodeTypeOK {
			return nil, fmt.Errorf("failure to broadcast tx sync: %s", res.RawLog)
		}
		results[i] = res
	}
	return results, nil
}
