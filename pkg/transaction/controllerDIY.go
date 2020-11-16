package transaction

import (
	"fmt"
	"time"

	"github.com/harmony-one/go-sdk/pkg/common"
	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/harmony/accounts"
	"github.com/harmony-one/harmony/accounts/keystore"
	"github.com/harmony-one/harmony/numeric"
)

// ExecuteTransactionDIY is the single entrypoint to execute a plain transaction.
// Each step in transaction creation, execution probably includes a mutation
// Each becomes a no-op if executionError occurred in any previous step
func (C *Controller) ExecuteTransactionDIY(
	nonce, gasLimit uint64,
	to string,
	shardID, toShardID uint32,
	amount, gasPrice numeric.Dec,
	inputData []byte,
) error {
	// WARNING Order of execution matters
	C.setShardIDs(shardID, toShardID)
	C.setIntrinsicGas(gasLimit)
	C.setGasPrice(gasPrice)
	C.setAmountDIY(amount)
	C.setReceiver(to)
	C.transactionForRPC.params["nonce"] = nonce
	C.setNewTransactionWithDataAndGas(inputData)
	switch C.Behavior.SigningImpl {
	case Software:
		C.signAndPrepareTxEncodedForSending()
	case Ledger:
		C.hardwareSignAndPrepareTxEncodedForSending()
	}
	// fmt.Println("Sending, nonce is:", C.transactionForRPC.params["nonce"])
	C.sendSignedTxDIY()
	// C.txConfirmation()
	return C.executionError
}

func (C *Controller) sendSignedTxDIY() {
	if C.executionError != nil || C.Behavior.DryRun {
		return
	}
	C.messengerDIY.SendRPCDIY(rpc.Method.SendRawTransaction, p{C.transactionForRPC.signature})
	// C.messenger.SendRPC(rpc.Method.SendRawTransaction, p{C.transactionForRPC.signature})
	// if err != nil {
	// 	C.executionError = err
	// 	return
	// }
	// r, _ := reply["result"].(string)
	// C.transactionForRPC.transactionHash = &r
}

// NewControllerDIY initializes a Controller, caller can control behavior via options
func NewControllerDIY(handler rpc.T,
	handlerDIY rpc.TDIY, senderKs *keystore.KeyStore,
	senderAcct *accounts.Account, chain common.ChainID,
	options ...func(*Controller),
) *Controller {
	txParams := make(map[string]interface{})
	ctrlr := &Controller{
		executionError: nil,
		messenger:      handler,
		messengerDIY:   handlerDIY,
		sender: sender{
			ks:      senderKs,
			account: senderAcct,
		},
		transactionForRPC: transactionForRPC{
			params:          txParams,
			signature:       nil,
			transactionHash: nil,
			receipt:         nil,
		},
		chain:    chain,
		Behavior: behavior{false, Software, 0},
	}
	for _, option := range options {
		option(ctrlr)
	}
	return ctrlr
}

func (C *Controller) setAmountDIY(amount numeric.Dec) {
	if C.executionError != nil {
		return
	}
	if amount.Sign() == -1 {
		C.executionError = ErrBadTransactionParam
		errorMsg := fmt.Sprintf(
			"can't set negative amount: %d", amount,
		)
		C.transactionErrors = append(C.transactionErrors, &Error{
			ErrMessage:           &errorMsg,
			TimestampOfRejection: time.Now().Unix(),
		})
		return
	}
	// balanceRPCReply, err := C.messenger.SendRPC(
	// 	rpc.Method.GetBalance,
	// 	p{address.ToBech32(C.sender.account.Address), "latest"},
	// )
	// if err != nil {
	// 	C.executionError = err
	// 	return
	// }
	// currentBalance, _ := balanceRPCReply["result"].(string)
	// bal, _ := new(big.Int).SetString(currentBalance[2:], 16)
	// balance := numeric.NewDecFromBigInt(bal)
	gasAsDec := C.transactionForRPC.params["gas-price"].(numeric.Dec)
	gasAsDec = gasAsDec.Mul(numeric.NewDec(int64(C.transactionForRPC.params["gas-limit"].(uint64))))
	amountInAtto := amount.Mul(oneAsDec)
	// total := amountInAtto.Add(gasAsDec)

	// if total.GT(balance) {
	// 	balanceInOne := balance.Quo(oneAsDec)
	// 	C.executionError = ErrBadTransactionParam
	// 	errorMsg := fmt.Sprintf(
	// 		"insufficient balance of %s in shard %d for the requested transfer of %s",
	// 		balanceInOne.String(), C.transactionForRPC.params["from-shard"].(uint32), amount.String(),
	// 	)
	// 	C.transactionErrors = append(C.transactionErrors, &Error{
	// 		ErrMessage:           &errorMsg,
	// 		TimestampOfRejection: time.Now().Unix(),
	// 	})
	// 	return
	// }
	C.transactionForRPC.params["transfer-amount"] = amountInAtto
}
