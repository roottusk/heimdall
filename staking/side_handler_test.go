package staking_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	ethTypes "github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/heimdall/app"
	"github.com/maticnetwork/heimdall/common"
	"github.com/maticnetwork/heimdall/contracts/stakinginfo"
	"github.com/maticnetwork/heimdall/helper/mocks"
	"github.com/maticnetwork/heimdall/staking"
	"github.com/maticnetwork/heimdall/staking/types"
	cmn "github.com/maticnetwork/heimdall/test"
	hmTypes "github.com/maticnetwork/heimdall/types"
	"github.com/maticnetwork/heimdall/types/simulation"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

//
// Create test suite
//

// SideHandlerTestSuite integrate test suite context object
type SideHandlerTestSuite struct {
	suite.Suite

	app            *app.HeimdallApp
	ctx            sdk.Context
	sideHandler    hmTypes.SideTxHandler
	postHandler    hmTypes.PostTxHandler
	contractCaller mocks.IContractCaller
	r              *rand.Rand
}

func (suite *SideHandlerTestSuite) SetupTest() {
	suite.app, suite.ctx, _ = createTestApp(false)

	suite.contractCaller = mocks.IContractCaller{}
	suite.sideHandler = staking.NewSideTxHandler(suite.app.StakingKeeper, &suite.contractCaller)
	suite.postHandler = staking.NewPostTxHandler(suite.app.StakingKeeper, &suite.contractCaller)

	// random generator
	s1 := rand.NewSource(time.Now().UnixNano())
	suite.r = rand.New(s1)
}

func TestSideHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(SideHandlerTestSuite))
}

//
// Test cases
//

func (suite *SideHandlerTestSuite) TestSideHandler() {
	t, ctx := suite.T(), suite.ctx

	// side handler
	result := suite.sideHandler(ctx, nil)
	require.Equal(t, uint32(sdk.CodeUnknownRequest), result.Code)
	require.Equal(t, abci.SideTxResultType_Skip, result.Result)
}

func (suite *SideHandlerTestSuite) TestSideHandleMsgValidatorJoin() {
	t, app, ctx := suite.T(), suite.app, suite.ctx
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	txHash := hmTypes.HexToHeimdallHash("123")
	index := simulation.RandIntBetween(r1, 0, 100)
	logIndex := uint64(index)
	validatorId := uint64(1)
	amount, _ := big.NewInt(0).SetString("1000000000000000000", 10)

	privKey1 := secp256k1.GenPrivKey()
	pubkey := hmTypes.NewPubKey(privKey1.PubKey().Bytes())
	address := pubkey.Address()

	chainParams := app.ChainKeeper.GetParams(ctx)
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(3)

	suite.Run("Success", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(1),
			sdk.NewInt(1000000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          address,
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.Equal(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should be success")
		require.Equal(t, abci.SideTxResultType_Yes, result.Result, "Result should be `yes`")
	})

	suite.Run("No receipt", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(1),
			sdk.NewInt(1000000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          address,
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(nil, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeWaitFrConfirmation), result.Code)
	})

	suite.Run("No EventLog", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(1),
			sdk.NewInt(1000000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(nil, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeErrDecodeEvent), result.Code)
	})

	suite.Run("Invalid Signer pubkey", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(1),
			sdk.NewInt(1000000000000000000),
			hmTypes.NewPubKey([]byte{234}),
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          address,
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid Signer address", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(1),
			sdk.NewInt(1000000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          hmTypes.ZeroHeimdallAddress.EthAddress(),
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid Validator Id", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			uint64(10),
			uint64(1),
			sdk.NewInt(1000000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          address,
			ValidatorId:     big.NewInt(1),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid Activation Epoch", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(10),
			sdk.NewInt(1000000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          address,
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid Amount", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(1),
			sdk.NewInt(100000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          address,
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid Block Number", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(1),
			sdk.NewInt(1000000000000000000),
			pubkey,
			txHash,
			logIndex,
			uint64(20),
			nonce.Uint64(),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          address,
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid nonce", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		msgValJoin := types.NewMsgValidatorJoin(
			hmTypes.BytesToHeimdallAddress(address.Bytes()),
			validatorId,
			uint64(1),
			sdk.NewInt(1000000000000000000),
			pubkey,
			txHash,
			logIndex,
			blockNumber.Uint64(),
			uint64(9),
		)

		stakinginfoStaked := &stakinginfo.StakinginfoStaked{
			Signer:          address,
			ValidatorId:     new(big.Int).SetUint64(validatorId),
			Nonce:           nonce,
			ActivationEpoch: big.NewInt(1),
			Amount:          amount,
			Total:           big.NewInt(10),
			SignerPubkey:    pubkey.Bytes()[1:],
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", txHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorJoinEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, msgValJoin.LogIndex).Return(stakinginfoStaked, nil)

		result := suite.sideHandler(ctx, msgValJoin)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})
}

func (suite *SideHandlerTestSuite) TestSideHandleMsgSingerUpdate() {
	t, app, ctx := suite.T(), suite.app, suite.ctx
	keeper := suite.app.StakingKeeper
	// pass 0 as time alive to generate non de-activated validators
	cmn.LoadValidatorSet(4, t, keeper, ctx, false, 0)
	oldValSet := keeper.GetValidatorSet(ctx)

	oldSigner := oldValSet.Validators[0]
	newSigner := cmn.GenRandomVal(1, 0, 10, 10, false, 1)
	newSigner[0].ID = oldSigner.ID
	newSigner[0].VotingPower = oldSigner.VotingPower
	chainParams := app.ChainKeeper.GetParams(ctx)
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(5)

	// gen msg
	msgTxHash := hmTypes.HexToHeimdallHash("123")

	suite.Run("Success", func() {
		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		suite.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := suite.sideHandler(ctx, msg)

		require.Equal(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should be success")
		require.Equal(t, abci.SideTxResultType_Yes, result.Result, "Result should be `yes`")
	})

	suite.Run("No Eventlog", func() {
		suite.contractCaller = mocks.IContractCaller{}

		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(nil, nil)

		result := suite.sideHandler(ctx, msg)

		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeErrDecodeEvent), result.Code)
	})

	suite.Run("Invalid BlockNumber", func() {
		suite.contractCaller = mocks.IContractCaller{}

		msg := types.NewMsgSignerUpdate(
			newSigner[0].Signer, uint64(oldSigner.ID),
			newSigner[0].PubKey,
			msgTxHash,
			0,
			uint64(9),
			nonce.Uint64(),
		)

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		suite.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := suite.sideHandler(ctx, msg)

		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid validator", func() {
		suite.contractCaller = mocks.IContractCaller{}

		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(6), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		suite.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := suite.sideHandler(ctx, msg)

		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid signer pubkey", func() {
		suite.contractCaller = mocks.IContractCaller{}

		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), hmTypes.NewPubKey([]byte{123}), msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		suite.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := suite.sideHandler(ctx, msg)

		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid new signer address", func() {
		suite.contractCaller = mocks.IContractCaller{}

		msg := types.NewMsgSignerUpdate(hmTypes.ZeroHeimdallAddress, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    hmTypes.ZeroHeimdallAddress.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		suite.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := suite.sideHandler(ctx, msg)

		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})

	suite.Run("Invalid nonce", func() {
		suite.contractCaller = mocks.IContractCaller{}

		msg := types.NewMsgSignerUpdate(newSigner[0].Signer, uint64(oldSigner.ID), newSigner[0].PubKey, msgTxHash, 0, blockNumber.Uint64(), uint64(12))

		txreceipt := &ethTypes.Receipt{BlockNumber: blockNumber}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		signerUpdateEvent := &stakinginfo.StakinginfoSignerChange{
			ValidatorId:  new(big.Int).SetUint64(oldSigner.ID.Uint64()),
			Nonce:        nonce,
			OldSigner:    oldSigner.Signer.EthAddress(),
			NewSigner:    newSigner[0].Signer.EthAddress(),
			SignerPubkey: newSigner[0].PubKey.Bytes()[1:],
		}
		suite.contractCaller.On("DecodeSignerUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(signerUpdateEvent, nil)

		result := suite.sideHandler(ctx, msg)

		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should Fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should be `skip`")
		require.Equal(t, uint32(common.CodeInvalidMsg), result.Code)
	})
}

func (suite *SideHandlerTestSuite) TestSideHandleMsgValidatorExit() {
	t, app, ctx := suite.T(), suite.app, suite.ctx
	keeper := app.StakingKeeper
	// pass 0 as time alive to generate non de-activated validators
	cmn.LoadValidatorSet(4, t, keeper, ctx, false, 0)
	validators := keeper.GetCurrentValidators(ctx)
	msgTxHash := hmTypes.HexToHeimdallHash("123")
	chainParams := app.ChainKeeper.GetParams(ctx)
	logIndex := uint64(0)
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(9)

	suite.Run("Success", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validators[0].Signer.EthAddress(),
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		suite.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

		msg := types.NewMsgValidatorExit(
			validators[0].Signer,
			uint64(validators[0].ID),
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		result := suite.sideHandler(ctx, msg)
		require.Equal(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should be success")
		require.Equal(t, abci.SideTxResultType_Yes, result.Result, "Result should be `yes`")
	})

	suite.Run("No Receipt", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(nil, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validators[0].Signer.EthAddress(),
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		suite.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

		msg := types.NewMsgValidatorExit(
			validators[0].Signer,
			uint64(validators[0].ID),
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("No Eventlog", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		validators[0].EndEpoch = 10

		suite.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(nil, nil)

		msg := types.NewMsgValidatorExit(
			validators[0].Signer,
			uint64(validators[0].ID),
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("Invalid BlockNumber", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validators[0].Signer.EthAddress(),
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		suite.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

		msg := types.NewMsgValidatorExit(
			validators[0].Signer,
			uint64(validators[0].ID),
			validators[0].EndEpoch,
			msgTxHash,
			0,
			uint64(5),
			nonce.Uint64(),
		)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})
	suite.Run("Invalid validatorId", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validators[0].Signer.EthAddress(),
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		suite.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

		msg := types.NewMsgValidatorExit(
			validators[0].Signer,
			uint64(66),
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("Invalid DeactivationEpoch", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validators[0].Signer.EthAddress(),
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}

		suite.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

		msg := types.NewMsgValidatorExit(
			validators[0].Signer,
			uint64(validators[0].ID),
			uint64(1000),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64(),
		)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("Invalid Nonce", func() {
		suite.contractCaller = mocks.IContractCaller{}
		txreceipt := &ethTypes.Receipt{
			BlockNumber: blockNumber,
		}

		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		amount, _ := big.NewInt(0).SetString("10000000000000000000", 10)
		stakinginfoUnstakeInit := &stakinginfo.StakinginfoUnstakeInit{
			User:              validators[0].Signer.EthAddress(),
			ValidatorId:       big.NewInt(0).SetUint64(validators[0].ID.Uint64()),
			Nonce:             nonce,
			DeactivationEpoch: big.NewInt(10),
			Amount:            amount,
		}
		validators[0].EndEpoch = 10

		suite.contractCaller.On("DecodeValidatorExitEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, logIndex).Return(stakinginfoUnstakeInit, nil)

		msg := types.NewMsgValidatorExit(
			validators[0].Signer,
			uint64(validators[0].ID),
			validators[0].EndEpoch,
			msgTxHash,
			0,
			blockNumber.Uint64(),
			uint64(6),
		)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})
}

func (suite *SideHandlerTestSuite) TestSideHandleMsgStakeUpdate() {
	t, app, ctx := suite.T(), suite.app, suite.ctx
	keeper := app.StakingKeeper

	// pass 0 as time alive to generate non de-activated validators
	cmn.LoadValidatorSet(4, t, keeper, ctx, false, 0)
	oldValSet := keeper.GetValidatorSet(ctx)
	oldVal := oldValSet.Validators[0]

	chainParams := app.ChainKeeper.GetParams(ctx)

	msgTxHash := hmTypes.HexToHeimdallHash("123")
	blockNumber := big.NewInt(10)
	nonce := big.NewInt(1)

	suite.Run("Success", func() {
		msg := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ID.Uint64(),
			sdk.NewInt(2000000000000000000),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}

		suite.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

		result := suite.sideHandler(ctx, msg)
		require.Equal(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should be success")
		require.Equal(t, abci.SideTxResultType_Yes, result.Result, "Result should be `yes`")
	})

	suite.Run("No Receipt", func() {
		suite.contractCaller = mocks.IContractCaller{}
		msg := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ID.Uint64(),
			sdk.NewInt(2000000000000000000),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(nil, nil)

		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}

		suite.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("No Eventlog", func() {
		suite.contractCaller = mocks.IContractCaller{}
		msg := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ID.Uint64(),
			sdk.NewInt(2000000000000000000),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		suite.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(nil, nil)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("Invalid BlockNumber", func() {
		suite.contractCaller = mocks.IContractCaller{}
		msg := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ID.Uint64(),
			sdk.NewInt(2000000000000000000),
			msgTxHash,
			0,
			uint64(15),
			nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}

		suite.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("Invalid ValidatorID", func() {
		suite.contractCaller = mocks.IContractCaller{}
		msg := types.NewMsgStakeUpdate(
			oldVal.Signer,
			uint64(13),
			sdk.NewInt(2000000000000000000),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}

		suite.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("Invalid Amount", func() {
		suite.contractCaller = mocks.IContractCaller{}
		msg := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ID.Uint64(),
			sdk.NewInt(200000000000000000),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			nonce.Uint64())

		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}

		suite.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})

	suite.Run("Invalid Nonce", func() {
		suite.contractCaller = mocks.IContractCaller{}
		msg := types.NewMsgStakeUpdate(
			oldVal.Signer,
			oldVal.ID.Uint64(),
			sdk.NewInt(2000000000000000000),
			msgTxHash,
			0,
			blockNumber.Uint64(),
			uint64(9))

		txreceipt := &ethTypes.Receipt{BlockNumber: big.NewInt(10)}
		suite.contractCaller.On("GetConfirmedTxReceipt", msgTxHash.EthHash(), chainParams.MainchainTxConfirmations).Return(txreceipt, nil)

		stakinginfoStakeUpdate := &stakinginfo.StakinginfoStakeUpdate{
			ValidatorId: new(big.Int).SetUint64(oldVal.ID.Uint64()),
			NewAmount:   new(big.Int).SetInt64(2000000000000000000),
			Nonce:       nonce,
		}

		suite.contractCaller.On("DecodeValidatorStakeUpdateEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), txreceipt, uint64(0)).Return(stakinginfoStakeUpdate, nil)

		result := suite.sideHandler(ctx, msg)
		require.NotEqual(t, uint32(sdk.CodeOK), result.Code, "Side tx handler should fail")
		require.Equal(t, abci.SideTxResultType_Skip, result.Result, "Result should skip")
	})
}

func (suite *SideHandlerTestSuite) TestPostHandler() {
	t, ctx := suite.T(), suite.ctx

	// post tx handler
	result := suite.postHandler(ctx, nil, abci.SideTxResultType_Yes)
	require.False(t, result.IsOK(), "Post handler should fail")
	require.Equal(t, sdk.CodeUnknownRequest, result.Code)
}

func (suite *SideHandlerTestSuite) TestpostHandleMsgValidatorJoin() {
}

func (suite *SideHandlerTestSuite) TestpostHandleMsgSingerUpdate() {
}

func (suite *SideHandlerTestSuite) TestpostHandleMsgValidatorExit() {
}
