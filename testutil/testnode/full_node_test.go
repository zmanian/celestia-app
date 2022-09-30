package testnode

import (
	"context"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-app/testutil"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/config"
	tmrand "github.com/tendermint/tendermint/libs/rand"
	"github.com/tendermint/tendermint/pkg/consts"
)

type IntegrationTestSuite struct {
	suite.Suite

	cleanups []func()
	accounts []string
	cctx     client.Context
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")
	require := s.Require()

	// we create an arbitray number of funded accounts
	for i := 0; i < 300; i++ {
		s.accounts = append(s.accounts, tmrand.Str(9))
	}

	tmCfg := config.DefaultConfig()
	tmCfg.Consensus.TimeoutCommit = time.Millisecond * 100
	tmNode, app, cctx, err := New(s.T(), tmCfg, false, s.accounts...)
	require.NoError(err)

	cctx, stopNode, err := StartNode(tmNode, cctx)
	require.NoError(err)
	s.cleanups = append(s.cleanups, stopNode)

	cctx, cleanupGRPC, err := StartGRPCServer(app, *DefaultAppConfig(), cctx)
	s.cleanups = append(s.cleanups, cleanupGRPC)

	s.cctx = cctx
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	for _, c := range s.cleanups {
		c()
	}
}

func (s *IntegrationTestSuite) Test_Liveness() {
	require := s.Require()
	err := WaitForNextBlock(s.cctx)
	require.NoError(err)
	// check that we're actually able to set the consensus params
	params, err := s.cctx.Client.ConsensusParams(context.TODO(), nil)
	require.NoError(err)
	require.Equal(int64(1), params.ConsensusParams.Block.TimeIotaMs)
	_, err = WaitForHeight(s.cctx, 20)
	require.NoError(err)
}

func (s *IntegrationTestSuite) Test_FillBlock() {
	require := s.Require()

	for squareSize := 16; squareSize < consts.MaxSquareSize; squareSize *= 2 {
		resps, err := FillBlock(s.cctx, squareSize, s.accounts)
		require.NoError(err)

		err = WaitForNextBlock(s.cctx)
		require.NoError(err)
		err = WaitForNextBlock(s.cctx)
		require.NoError(err)

		var inclusionHeight int64
		for _, v := range resps {
			res, err := testutil.QueryWithOutProof(s.cctx, v.TxHash)
			require.NoError(err)
			require.Equal(abci.CodeTypeOK, res.TxResult.Code)
			if inclusionHeight == 0 {
				inclusionHeight = res.Height
				continue
			}
			// check that all of the txs are included in the same block
			require.Equal(inclusionHeight, res.Height)
		}

		b, err := s.cctx.Client.Block(context.TODO(), &inclusionHeight)
		require.NoError(err)
		require.Equal(uint64(squareSize), b.Block.OriginalSquareSize)
	}

}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
