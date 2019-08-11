package types

import (
	"fmt"

	"github.com/maticnetwork/heimdall/types"
)

// NodeQuerier is an interface that is satisfied by types that provide the QueryWithData method
type NodeQuerier interface {
	// QueryWithData performs a query to a Tendermint node with the provided path
	// and a data payload. It returns the result and height of the query upon success
	// or an error if the query fails.
	QueryWithData(path string, data []byte) ([]byte, error)
}

// AccountRetriever defines the properties of a type that can be used to
// retrieve accounts.
type AccountRetriever struct {
	querier NodeQuerier
}

// NewAccountRetriever initialises a new AccountRetriever instance.
func NewAccountRetriever(querier NodeQuerier) AccountRetriever {
	return AccountRetriever{querier: querier}
}

// GetAccount queries for an account given an address and a block height. An
// error is returned if the query or decoding fails.
func (ar AccountRetriever) GetAccount(addr types.HeimdallAddress) (Account, error) {
	account, err := ar.GetAccountWithHeight(addr)
	return account, err
}

// GetAccountWithHeight queries for an account given an address. Returns the
// height of the query with the account. An error is returned if the query
// or decoding fails.
func (ar AccountRetriever) GetAccountWithHeight(addr types.HeimdallAddress) (Account, error) {
	bs, err := MsgCdc.MarshalJSON(NewQueryAccountParams(addr))
	if err != nil {
		return nil, err
	}

	res, err := ar.querier.QueryWithData(fmt.Sprintf("custom/%s/%s", QuerierRoute, QueryAccount), bs)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := MsgCdc.UnmarshalJSON(res, &account); err != nil {
		return nil, err
	}

	return account, nil
}

// EnsureExists returns an error if no account exists for the given address else nil.
func (ar AccountRetriever) EnsureExists(addr types.HeimdallAddress) error {
	if _, err := ar.GetAccount(addr); err != nil {
		return err
	}
	return nil
}

// GetAccountNumberSequence returns sequence and account number for the given address.
// It returns an error if the account couldn't be retrieved from the state.
func (ar AccountRetriever) GetAccountNumberSequence(addr types.HeimdallAddress) (uint64, uint64, error) {
	acc, err := ar.GetAccount(addr)
	if err != nil {
		return 0, 0, err
	}
	return acc.GetAccountNumber(), acc.GetSequence(), nil
}
