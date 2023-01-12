package testnode

import (
	"context"
	"fmt"
	"testing"

	"github.com/celestiaorg/celestia-app/app"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	tmrand "github.com/tendermint/tendermint/libs/rand"
)

type EvanAndMattDebugSuite struct {
	suite.Suite

	cleanups []func() error
	accounts []string
	cctx     Context
}

func (s *EvanAndMattDebugSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("skipping full node integration test in short mode.")
	}

	s.T().Log("setting up integration test suite")
	require := s.Require()

	// we create an arbitrary number of funded accounts
	for i := 0; i < 300; i++ {
		s.accounts = append(s.accounts, tmrand.Str(9))
	}

	tmNode, app, cctx, err := New(s.T(), DefaultParams(), DefaultTendermintConfig(), false, s.accounts...)
	require.NoError(err)

	cctx, stopNode, err := StartNode(tmNode, cctx)
	require.NoError(err)
	s.cleanups = append(s.cleanups, stopNode)

	cctx, cleanupGRPC, err := StartGRPCServer(app, DefaultAppConfig(), cctx)
	require.NoError(err)
	s.cleanups = append(s.cleanups, cleanupGRPC)

	s.cctx = cctx
}

func (s *EvanAndMattDebugSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	for _, c := range s.cleanups {
		err := c()
		require.NoError(s.T(), err)
	}
}

func (s *EvanAndMattDebugSuite) TestQueryAccount() {
	t := s.T()
	bankclient := banktypes.NewQueryClient(s.cctx.GRPCClient)

	acc := s.accounts[0]
	rec, err := s.cctx.Keyring.Key(acc)
	require.NoError(t, err)

	addr, err := rec.GetAddress()
	require.NoError(t, err)

	resp, err := bankclient.Balance(context.TODO(), banktypes.NewQueryBalanceRequest(addr, app.BondDenom))
	require.NoError(t, err)

	fmt.Println(resp.Balance)
}

func TestEvanAndMattDebugSuite(t *testing.T) {
	suite.Run(t, new(EvanAndMattDebugSuite))
}
