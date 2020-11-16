package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harmony-one/go-sdk/pkg/address"
	"github.com/harmony-one/go-sdk/pkg/common"
	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/go-sdk/pkg/transaction"
	"github.com/harmony-one/harmony/accounts"
	"github.com/harmony-one/harmony/accounts/keystore"
	"github.com/harmony-one/harmony/core"
	"github.com/pkg/errors"
)

// handlerForBulkTransactions checks and sets all flags for a transaction
// from the element at index of transferFileFlags, then calls handlerForTransaction.
func handlerForBulkTransactionsDIY(txLog *transactionLog, index int, ks *keystore.KeyStore, acct *accounts.Account) error {
	txnFlags := transferFileFlags[index]

	// Check for required fields.
	if txnFlags.FromAddress == nil || txnFlags.ToAddress == nil || txnFlags.Amount == nil {
		return handlerForError(txLog, errors.New("FromAddress/ToAddress/Amount are required fields"))
	}
	if txnFlags.FromShardID == nil || txnFlags.ToShardID == nil {
		return handlerForError(txLog, errors.New("FromShardID/ToShardID are required fields"))
	}

	// Set required fields.
	var fromAddLocal oneAddress
	err := fromAddLocal.Set(*txnFlags.FromAddress)
	if handlerForError(txLog, err) != nil {
		return err
	}

	var toAddLocal oneAddress
	err = toAddLocal.Set(*txnFlags.ToAddress)
	if handlerForError(txLog, err) != nil {
		return err
	}

	amountLocal := *txnFlags.Amount

	fromShard, err := strconv.ParseUint(*txnFlags.FromShardID, 10, 64)
	if handlerForError(txLog, err) != nil {
		return err
	}
	fromShardIDLocal := uint32(fromShard)

	toShard, err := strconv.ParseUint(*txnFlags.ToShardID, 10, 64)
	if handlerForError(txLog, err) != nil {
		return err
	}
	toShardIDLocal := uint32(toShard)

	// Set optional fields.
	if txnFlags.PassphraseFile != nil {
		passphraseFilePath = *txnFlags.PassphraseFile
		passphrase, err = getPassphrase()
		if handlerForError(txLog, err) != nil {
			return err
		}
	} else if txnFlags.PassphraseString != nil {
		passphrase = *txnFlags.PassphraseString
	} else {
		passphrase = common.DefaultPassphrase
	}

	var inputNonceLocal string
	if txnFlags.InputNonce != nil {
		inputNonceLocal = *txnFlags.InputNonce
	} else {
		inputNonceLocal = "" // Reset to default for subsequent transactions
	}

	if txnFlags.GasPrice != nil {
		gasPrice = *txnFlags.GasPrice
	} else {
		gasPrice = "1" // Reset to default for subsequent transactions
	}
	if txnFlags.GasLimit != nil {
		gasLimit = *txnFlags.GasLimit
	} else {
		gasLimit = "" // Reset to default for subsequent transactions
	}
	trueNonce = txnFlags.TrueNonce

	return handlerForTransactionDIY(txLog, fromAddLocal, toAddLocal, fromShardIDLocal, toShardIDLocal,
		amountLocal, inputNonceLocal, ks, acct)
}

// handlerForTransaction executes a single transaction and fills out the transaction logger accordingly.
//
// Note that the vars need to be set before calling this handler.
func handlerForTransactionDIY(txLog *transactionLog, fromAddLocal oneAddress, toAddLocal oneAddress,
	fromShardIDLocal uint32, toShardIDLocal uint32, amountLocal string, inputNonceLocal string,
	ks *keystore.KeyStore, acct *accounts.Account) error {
	from := fromAddLocal.String()
	// s, err := sharding.Structure(node)
	// if handlerForError(txLog, err) != nil {
	// 	return err
	// }
	// err := validation.ValidShardIDs(fromShardIDLocal, toShardIDLocal, uint32(len(s)))
	// if handlerForError(txLog, err) != nil {
	// 	return err
	// }
	networkHandler, err := handlerForShard(fromShardIDLocal, node)
	if handlerForError(txLog, err) != nil {
		return err
	}

	var ctrlr *transaction.Controller
	if useLedgerWallet {
		account := accounts.Account{Address: address.Parse(from)}
		ctrlr = transaction.NewControllerDIY(networkHandler, networkHandler, nil, &account, *chainName.chainID, opts)
	} else {
		// 这个语句开goroutine多了会有问题
		// ks, acct, err := store.UnlockedKeystore(from, passphrase)
		if handlerForError(txLog, err) != nil {
			return err
		}
		ctrlr = transaction.NewControllerDIY(networkHandler, networkHandler, ks, acct, *chainName.chainID, opts)
	}

	nonce, err := getNonceDIY(fromAddLocal.String(), networkHandler, inputNonceLocal)
	if err != nil {
		return err
	}

	amt, err := common.NewDecFromString(amountLocal)
	if err != nil {
		amtErr := fmt.Errorf("amount %w", err)
		handlerForError(txLog, amtErr)
		return amtErr
	}

	gPrice, err := common.NewDecFromString(gasPrice)
	if err != nil {
		gasErr := fmt.Errorf("gas-price %w", err)
		handlerForError(txLog, gasErr)
		return gasErr
	}

	// 给交易增加负载以增加交易大小
	size := make([]byte, 300)

	var gLimit uint64
	if gasLimit == "" {
		gLimit, err = core.IntrinsicGas(size, false, true, false)
		if handlerForError(txLog, err) != nil {
			return err
		}
	} else {
		if strings.HasPrefix(gasLimit, "-") {
			limitErr := errors.New(fmt.Sprintf("gas-limit can not be negative: %s", gasLimit))
			handlerForError(txLog, limitErr)
			return limitErr
		}
		tempLimit, e := strconv.ParseInt(gasLimit, 10, 64)
		if handlerForError(txLog, e) != nil {
			return e
		}
		gLimit = uint64(tempLimit)
	}

	txLog.TimeSigned = time.Now().UTC().Format(timeFormat) // Approximate time of signature

	// go func() {
	// 	err = ctrlr.ExecuteTransactionDIY(
	// 		nonce, gLimit,
	// 		toAddress.String(),
	// 		fromShardID, toShardID,
	// 		amt, gPrice,
	// 		[]byte{},
	// 	)
	// 	wg.Done()
	// }()

	err = ctrlr.ExecuteTransactionDIY(
		nonce, gLimit,
		toAddLocal.String(),
		fromShardIDLocal, toShardIDLocal,
		amt, gPrice,
		size,
	)

	if dryRun {
		txLog.RawTxn = ctrlr.RawTransaction()
		txLog.Transaction = make(map[string]interface{})
		_ = json.Unmarshal([]byte(ctrlr.TransactionToJSON(false)), &txLog.Transaction)
	}
	// else if txHash := ctrlr.TransactionHash(); txHash != nil {
	// 	txLog.TxHash = *txHash
	// }
	// txLog.Receipt = ctrlr.Receipt()["result"]
	if err != nil {
		// Report all transaction errors first...
		for _, txError := range ctrlr.TransactionErrors() {
			_ = handlerForError(txLog, txError.Error())
		}
		err = handlerForError(txLog, err)
	}
	// if !dryRun && timeout > 0 && txLog.Receipt == nil {
	// 	err = handlerForError(txLog, errors.New("Failed to confirm transaction"))
	// }
	return err
}

func getNonceDIY(address string, messenger rpc.T, inputNonceLocal string) (uint64, error) {
	if trueNonce {
		// cannot define nonce when using true nonce
		return transaction.GetNextNonce(address, messenger), nil
	}
	return getNonceFromInputDIY(address, inputNonceLocal, messenger)
}

func getNonceFromInputDIY(addr, inputNonceLocal string, messenger rpc.T) (uint64, error) {
	if inputNonceLocal != "" {
		if strings.HasPrefix(inputNonceLocal, "-") {
			return 0, errors.New(fmt.Sprintf("nonce can not be negative: %s", inputNonceLocal))
		}
		nonce, err := strconv.ParseUint(inputNonceLocal, 10, 64)
		if err != nil {
			return 0, err
		} else {
			return nonce, nil
		}
	} else {
		return transaction.GetNextPendingNonce(addr, messenger), nil
	}
}
