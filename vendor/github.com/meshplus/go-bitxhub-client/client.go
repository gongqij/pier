package rpcx

import (
	"context"

	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/types"
	"github.com/meshplus/bitxhub-model/pb"
)

type SubscriptionType int

const (
	SubscribeNewBlock SubscriptionType = iota
)

//go:generate mockgen -destination mock_client/mock_client.go -package mock_client -source client.go
type Client interface {
	//Close all connections between BitXHub and the client.
	Stop() error

	//Reset ecdsa key.
	SetPrivateKey(crypto.PrivateKey)

	//Send a readonly transaction to BitXHub. If the transaction is writable,
	// this transaction will not be executed and error wil be returned.
	SendView(tx *pb.BxhTransaction) (*pb.Receipt, error)

	//Send a signed transaction to BitXHub. If the signature is illegal,
	//the transaction hash will be obtained but the transaction receipt is illegal.
	SendTransaction(tx *pb.BxhTransaction, opts *TransactOpts) (string, error)

	//Send transaction to BitXHub and get the receipt.
	SendTransactionWithReceipt(tx *pb.BxhTransaction, opts *TransactOpts) (*pb.Receipt, error)

	//Get the receipt by transaction hash,
	//the status of the receipt is a sign of whether the transaction is successful.
	GetReceipt(hash string) (*pb.Receipt, error)

	//Get transaction from BitXHub by transaction hash.
	GetTransaction(hash string) (*pb.GetTransactionResponse, error)

	//Get the current blockchain situation of BitXHub.
	GetChainMeta() (*pb.ChainMeta, error)

	//Get blocks of the specified block height range.
	GetBlocks(start uint64, end uint64) (*pb.GetBlocksResponse, error)

	//Obtain block information from BitXHub.
	//The block header contains the basic information of the block,
	//and the block body contains all the transactions packaged.
	GetBlock(value string, blockType pb.GetBlockRequest_Type) (*pb.Block, error)

	//Get the status of the blockchain from BitXHub, normal or abnormal.
	GetChainStatus() (*pb.Response, error)

	//Get the validators from BitXHub.
	GetValidators() (*pb.Response, error)

	//Get the current network situation of BitXHub.
	GetNetworkMeta() (*pb.Response, error)

	//Get account balance from BitXHub by address.
	GetAccountBalance(address string) (*pb.Response, error)

	//Get the missing block header from BitXHub.
	GetBlockHeader(ctx context.Context, begin, end uint64, ch chan<- *pb.BlockHeader) error

	//Get the missing block header from BitXHub.
	GetInterchainTxWrappers(ctx context.Context, pid string, begin, end uint64, ch chan<- *pb.InterchainTxWrappers) error

	//Subscribe to event notifications from BitXHub.
	Subscribe(context.Context, pb.SubscriptionRequest_Type, []byte) (<-chan interface{}, error)

	//SubscribeAudit to event notifications from BitXHub with permission.
	SubscribeAudit(context.Context, pb.AuditSubscriptionRequest_Type, uint64, []byte) (<-chan interface{}, error)

	//Deploy the contract, the contract address will be returned when the deployment is successful.
	DeployContract(contract []byte, opts *TransactOpts) (contractAddr *types.Address, err error)

	//GenerateContractTx generates signed transaction to invoke contract
	GenerateContractTx(vmType pb.TransactionData_VMType, address *types.Address, method string, args ...*pb.Arg) (*pb.BxhTransaction, error)

	// GenerateIBTPTx generates interchain tx with ibtp specified
	GenerateIBTPTx(ibtp *pb.IBTP) (*pb.BxhTransaction, error)

	//Call the contract according to the contract type, contract address,
	//contract method, and contract method parameters
	InvokeContract(vmType pb.TransactionData_VMType, address *types.Address, method string, opts *TransactOpts, args ...*pb.Arg) (*pb.Receipt, error)

	//Invoke the BVM contract, BVM is BitXHub's blot contract.
	InvokeBVMContract(address *types.Address, method string, opts *TransactOpts, args ...*pb.Arg) (*pb.Receipt, error)

	//Invoke the XVM contract, XVM is WebAssembly contract.
	InvokeXVMContract(address *types.Address, method string, opts *TransactOpts, args ...*pb.Arg) (*pb.Receipt, error)

	// Get BitXHub's signatures specified by id and type.
	GetMultiSigns(id string, typ pb.GetSignsRequest_Type) (*pb.SignResponse, error)

	// Get BitXHub's tss signatures specified by id and type.
	GetTssSigns(id string, typ pb.GetSignsRequest_Type, extra []byte) (*pb.SignResponse, error)

	// Get BitXHub TPS during block [begin, end]
	GetTPS(begin, end uint64) (uint64, error)

	// GetPendingNonceByAccount returns the latest nonce of an account in the pending status,
	// and it should be the nonce for next transaction
	GetPendingNonceByAccount(account string) (uint64, error)

	// IPFSPutFromLocal puts local file to ipfs network
	IPFSPutFromLocal(localfPath string) (*pb.Response, error)

	// IPFSGet gets from ipfs network
	IPFSGet(path string) (*pb.Response, error)

	// IPFSGetToLocal gets from ipfs and saves to local file path
	IPFSGetToLocal(path string, localfPath string) (*pb.Response, error)
	//Check whethe there is a master pier connect to the BitXHub.
	CheckMasterPier(address string) (*pb.Response, error)

	//Set the master pier connect to the BitXHub.
	SetMasterPier(address string, index string, timeout int64) (*pb.Response, error)

	//Update the master pier status
	HeartBeat(address string, index string) (*pb.Response, error)

	// GetChainID get BitXHub Chain ID
	GetChainID() (uint64, error)
}

type TransactOpts struct {
	From  string
	Nonce uint64
}
