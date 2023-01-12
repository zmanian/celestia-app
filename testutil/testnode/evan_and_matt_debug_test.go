package testnode

import (
	"fmt"
	"testing"

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

	genState, kr, err := DefaultGenesisState(s.accounts...)
	require.NoError(err)

	tmNode, app, cctx, err := New(s.T(), DefaultParams(), DefaultTendermintConfig(), false, genState, kr)
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
	height := int64(1)
	blockRes, err := s.cctx.Client.Block(s.cctx.GoContext(), &height)
	require.NoError(t, err)
	fmt.Println(blockRes.Block.ChainID, len(blockRes.Block.Txs))
}

func TestEvanAndMattDebugSuite(t *testing.T) {
	suite.Run(t, new(EvanAndMattDebugSuite))
}
