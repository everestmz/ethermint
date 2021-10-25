package network

import (
	"math/big"
	"time"

	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"

	"github.com/cosmos/cosmos-sdk/simapp"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/tharsis/ethermint/app"
	"github.com/tharsis/ethermint/crypto/ethsecp256k1"
	"github.com/tharsis/ethermint/tests"
	ethermint "github.com/tharsis/ethermint/types"
	"github.com/tharsis/ethermint/x/evm"
	"github.com/tharsis/ethermint/x/evm/types"

	"github.com/tendermint/tendermint/crypto/tmhash"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	feemarkettypes "github.com/tharsis/ethermint/x/feemarket/types"

	"github.com/tendermint/tendermint/version"
)

func FuzzEVMHandlerImpl(amount1 int64, gasLimit1 uint64, gasPrice1 int64, input1 []byte,
	amount2 int64, nonce2 uint64, gasLimit2 uint64, gasPrice2 int64, input2 []byte,
	amount3 int64, gasLimit3 uint64, gasPrice3 int64, input3 []byte) error {

	checkTx := false

	// account key
	priv, err := ethsecp256k1.GenerateKey()
	if err != nil {
		return err
	}
	address := common.BytesToAddress(priv.PubKey().Address().Bytes())
	signer := tests.NewSigner(priv)
	from := address
	// consensus key
	priv, err = ethsecp256k1.GenerateKey()
	if err != nil {
		return err
	}
	consAddress := sdk.ConsAddress(priv.PubKey().Address())

	eapp := app.Setup(checkTx, &feemarkettypes.GenesisState{})
	coins := sdk.NewCoins(sdk.NewCoin(types.DefaultEVMDenom, sdk.NewInt(100000000000000)))
	genesisState := app.ModuleBasics.DefaultGenesis(eapp.AppCodec())
	b32address := sdk.MustBech32ifyAddressBytes(sdk.GetConfig().GetBech32AccountAddrPrefix(), priv.PubKey().Address().Bytes())
	balances := []banktypes.Balance{
		{
			Address: b32address,
			Coins:   coins,
		},
		{
			Address: eapp.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName).String(),
			Coins:   coins,
		},
	}
	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, sdk.NewCoins(sdk.NewCoin(types.DefaultEVMDenom, sdk.NewInt(200000000000000))), []banktypes.Metadata{})
	genesisState[banktypes.ModuleName] = eapp.AppCodec().MustMarshalJSON(bankGenesis)

	stateBytes, err := tmjson.MarshalIndent(genesisState, "", " ")
	if err != nil {
		return err
	}

	// Initialize the chain
	eapp.InitChain(
		abci.RequestInitChain{
			ChainId:         "ethermint_9000-1",
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: simapp.DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)

	ctx := eapp.BaseApp.NewContext(checkTx, tmproto.Header{
		Height:          1,
		ChainID:         "ethermint_9000-1",
		Time:            time.Now().UTC(),
		ProposerAddress: consAddress.Bytes(),
		Version: tmversion.Consensus{
			Block: version.BlockProtocol,
		},
		LastBlockId: tmproto.BlockID{
			Hash: tmhash.Sum([]byte("block_id")),
			PartSetHeader: tmproto.PartSetHeader{
				Total: 11,
				Hash:  tmhash.Sum([]byte("partset_header")),
			},
		},
		AppHash:            tmhash.Sum([]byte("app")),
		DataHash:           tmhash.Sum([]byte("data")),
		EvidenceHash:       tmhash.Sum([]byte("evidence")),
		ValidatorsHash:     tmhash.Sum([]byte("validators")),
		NextValidatorsHash: tmhash.Sum([]byte("next_validators")),
		ConsensusHash:      tmhash.Sum([]byte("consensus")),
		LastResultsHash:    tmhash.Sum([]byte("last_result")),
	})
	eapp.EvmKeeper.WithContext(ctx)

	queryHelper := baseapp.NewQueryServerTestHelper(ctx, eapp.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, eapp.EvmKeeper)

	acc := &ethermint.EthAccount{
		BaseAccount: authtypes.NewBaseAccount(sdk.AccAddress(address.Bytes()), nil, 0, 0),
		CodeHash:    common.BytesToHash(crypto.Keccak256(nil)).String(),
	}

	eapp.AccountKeeper.SetAccount(ctx, acc)

	valAddr := sdk.ValAddress(address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr, priv.PubKey(), stakingtypes.Description{})
	if err != nil {
		return err
	}

	err = eapp.StakingKeeper.SetValidatorByConsAddr(ctx, validator)
	if err != nil {
		return err
	}
	err = eapp.StakingKeeper.SetValidatorByConsAddr(ctx, validator)
	if err != nil {
		return err
	}
	eapp.StakingKeeper.SetValidator(ctx, validator)

	ethSigner := ethtypes.LatestSignerForChainID(eapp.EvmKeeper.ChainID())
	handler := evm.NewHandler(eapp.EvmKeeper)

	to := crypto.CreateAddress(from, 1)
	chainID := big.NewInt(1)
	tx1 := types.NewTxContract(chainID, 0, big.NewInt(amount1), gasLimit1, big.NewInt(gasPrice1), nil, nil, input1, nil)
	tx1.From = from.String()
	tx1.Sign(ethSigner, signer)

	tx2 := types.NewTx(chainID, nonce2, &to, big.NewInt(amount2), gasLimit2, big.NewInt(gasPrice2), nil, nil, input2, nil)
	tx2.From = from.String()

	tx2.Sign(ethSigner, signer)

	tx3 := types.NewTx(chainID, 1, &to, big.NewInt(amount3), gasLimit3, big.NewInt(gasPrice3), nil, nil, input3, nil)
	tx3.From = from.String()

	tx3.Sign(ethSigner, signer)
	handler(ctx, tx1)
	handler(ctx, tx2)
	handler(ctx, tx3)

	return nil
}
