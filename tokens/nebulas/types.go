package nebulas

import (
	"encoding/json"
	"errors"
)

const (
	TxPayloadBinaryType = "binary"
	TxPayloadDeployType = "deploy"
	TxPayloadCallType   = "call"
)

// Error Types
var (
	ErrInvalidBlockOnCanonicalChain                      = errors.New("invalid block, it's not on canonical chain")
	ErrNotBlockInCanonicalChain                          = errors.New("cannot find the block in canonical chain")
	ErrInvalidBlockCannotFindParentInLocal               = errors.New("invalid block received, download its parent from others")
	ErrCannotFindBlockAtGivenHeight                      = errors.New("cannot find a block at given height which is less than tail block's height")
	ErrInvalidBlockCannotFindParentInLocalAndTryDownload = errors.New("invalid block received, download its parent from others")
	ErrInvalidBlockCannotFindParentInLocalAndTrySync     = errors.New("invalid block received, sync its parent from others")
	ErrBlockNotFound                                     = errors.New("block not found in blockchain cache nor chain")

	ErrInvalidConfigChainID          = errors.New("invalid chainID, genesis chainID not equal to chainID in config")
	ErrCannotLoadGenesisConf         = errors.New("cannot load genesis conf")
	ErrGenesisNotEqualChainIDInDB    = errors.New("Failed to check. genesis chainID not equal in db")
	ErrGenesisNotEqualDynastyInDB    = errors.New("Failed to check. genesis dynasty not equal in db")
	ErrGenesisNotEqualTokenInDB      = errors.New("Failed to check. genesis TokenDistribution not equal in db")
	ErrGenesisNotEqualDynastyLenInDB = errors.New("Failed to check. genesis dynasty length not equal in db")
	ErrGenesisNotEqualTokenLenInDB   = errors.New("Failed to check. genesis TokenDistribution length not equal in db")

	ErrLinkToWrongParentBlock = errors.New("link the block to a block who is not its parent")
	ErrMissingParentBlock     = errors.New("cannot find the block's parent block in storage")
	ErrInvalidBlockHash       = errors.New("invalid block hash")
	ErrDoubleSealBlock        = errors.New("cannot seal a block twice")
	ErrDuplicatedBlock        = errors.New("duplicated block")
	ErrDoubleBlockMinted      = errors.New("double block minted")
	ErrVRFProofFailed         = errors.New("VRF proof failed")
	ErrInvalidBlockRandom     = errors.New("invalid block random")
	ErrInvalidBlockProposer   = errors.New("invalid block proposer")

	ErrInvalidChainID           = errors.New("invalid transaction chainID")
	ErrInvalidTransactionSigner = errors.New("invalid transaction signer")
	ErrInvalidTransactionHash   = errors.New("invalid transaction hash")
	ErrInvalidSignature         = errors.New("invalid transaction signature")
	ErrInvalidTxPayloadType     = errors.New("invalid transaction data payload type")
	ErrInvalidGasPrice          = errors.New("invalid gas price, should be in (0, 10^12]")
	ErrInvalidGasLimit          = errors.New("invalid gas limit, should be in (0, 5*10^10]")

	ErrNoTimeToPackTransactions       = errors.New("no time left to pack transactions in a block")
	ErrTxDataPayLoadOutOfMaxLength    = errors.New("data's payload is out of max data length")
	ErrTxDataBinPayLoadOutOfMaxLength = errors.New("data's payload is out of max data length in a binary tx")
	ErrNilArgument                    = errors.New("argument(s) is nil")
	ErrInvalidArgument                = errors.New("invalid argument(s)")

	ErrInsufficientBalance                = errors.New("insufficient balance")
	ErrBelowGasPrice                      = errors.New("below the gas price")
	ErrGasCntOverflow                     = errors.New("the count of gas used is overflow")
	ErrGasFeeOverflow                     = errors.New("the fee of gas used is overflow")
	ErrInvalidTransfer                    = errors.New("transfer error: overflow or insufficient balance")
	ErrGasLimitLessOrEqualToZero          = errors.New("gas limit less or equal to 0")
	ErrOutOfGasLimit                      = errors.New("out of gas limit")
	ErrTxExecutionFailed                  = errors.New("transaction execution failed")
	ErrZeroGasPrice                       = errors.New("gas price should be greater than zero")
	ErrZeroGasLimit                       = errors.New("gas limit should be greater than zero")
	ErrContractDeployFailed               = errors.New("contract deploy failed")
	ErrContractCheckFailed                = errors.New("contract check failed")
	ErrContractTransactionAddressNotEqual = errors.New("contract transaction from-address not equal to to-address")

	ErrDuplicatedTransaction = errors.New("duplicated transaction")
	ErrSmallTransactionNonce = errors.New("cannot accept a transaction with smaller nonce")
	ErrLargeTransactionNonce = errors.New("cannot accept a transaction with too bigger nonce")

	ErrInvalidAddress         = errors.New("address: invalid address")
	ErrInvalidAddressFormat   = errors.New("address: invalid address format")
	ErrInvalidAddressType     = errors.New("address: invalid address type")
	ErrInvalidAddressChecksum = errors.New("address: invalid address checksum")

	ErrInvalidCandidatePayloadAction     = errors.New("invalid transaction candidate payload action")
	ErrInvalidDelegatePayloadAction      = errors.New("invalid transaction vote payload action")
	ErrInvalidDelegateToNonCandidate     = errors.New("cannot delegate to non-candidate")
	ErrInvalidUnDelegateFromNonDelegatee = errors.New("cannot un-delegate from non-delegatee")

	ErrCloneWorldState           = errors.New("Failed to clone world state")
	ErrCloneAccountState         = errors.New("Failed to clone account state")
	ErrCloneTxsState             = errors.New("Failed to clone txs state")
	ErrCloneEventsState          = errors.New("Failed to clone events state")
	ErrInvalidBlockStateRoot     = errors.New("invalid block state root hash")
	ErrInvalidBlockTxsRoot       = errors.New("invalid block txs root hash")
	ErrInvalidBlockEventsRoot    = errors.New("invalid block events root hash")
	ErrInvalidBlockConsensusRoot = errors.New("invalid block consensus root hash")
	ErrInvalidProtoToBlock       = errors.New("protobuf message cannot be converted into Block")
	ErrInvalidProtoToBlockHeader = errors.New("protobuf message cannot be converted into BlockHeader")
	ErrInvalidProtoToTransaction = errors.New("protobuf message cannot be converted into Transaction")
	ErrInvalidTransactionData    = errors.New("invalid data in tx from Proto")
	ErrInvalidDagBlock           = errors.New("block's dag is incorrect")

	ErrCannotRevertLIB        = errors.New("cannot revert latest irreversible block")
	ErrCannotLoadGenesisBlock = errors.New("cannot load genesis block from storage")
	ErrCannotLoadLIBBlock     = errors.New("cannot load tail block from storage")
	ErrCannotLoadTailBlock    = errors.New("cannot load latest irreversible block from storage")
	ErrGenesisConfNotMatch    = errors.New("Failed to load genesis from storage, different with genesis conf")

	ErrInvalidDeploySource     = errors.New("invalid source of deploy payload")
	ErrInvalidDeploySourceType = errors.New("invalid source type of deploy payload")
	ErrInvalidCallFunction     = errors.New("invalid function of call payload")

	ErrInvalidTransactionResultEvent  = errors.New("invalid transaction result event, the last event in tx's events should be result event")
	ErrNotFoundTransactionResultEvent = errors.New("transaction result event is not found ")

	// nvm error
	ErrExecutionFailed = errors.New("execution failed")
	ErrUnexpected      = errors.New("Unexpected sys error")
	// multi nvm error
	ErrInnerExecutionFailed = errors.New("multi execution failed")
	ErrCreateInnerTx        = errors.New("Failed to create inner transaction")

	// access control
	ErrUnsupportedKeyword      = errors.New("transaction data has unsupported keyword")
	ErrUnsupportedFunction     = errors.New("transaction payload has unsupported function")
	ErrRestrictedFromAddress   = errors.New("transaction from address is restricted")
	ErrRestrictedToAddress     = errors.New("transaction to address is restricted")
	ErrNrc20ArgsCheckFailed    = errors.New("transaction nrc20 args check failed")
	ErrNrc20AddressCheckFailed = errors.New("transaction nrc20 address check failed")
	ErrNrc20ValueCheckFailed   = errors.New("transaction nrc20 value check failed")

	// func deprecated
	ErrFuncDeprecated = errors.New("function deprecated")

	ErrBlockStateCheckFailed = errors.New("Failed to check block state")
)

type NebResponse struct {
	Result GetNebStateResponse `json:"result"`
}

type GetNebStateResponse struct {
	// Block chain id
	ChainId uint32 `protobuf:"varint,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	// Current neb tail hash
	Tail string `protobuf:"bytes,2,opt,name=tail,proto3" json:"tail,omitempty"`
	// Current neb lib hash
	Lib string `protobuf:"bytes,3,opt,name=lib,proto3" json:"lib,omitempty"`
	// Current neb tail block height
	Height uint64 `protobuf:"varint,4,opt,name=height,proto3" json:"height,omitempty,string"`
	// The current neb protocol version.
	ProtocolVersion string `protobuf:"bytes,6,opt,name=protocol_version,json=protocolVersion,proto3" json:"protocol_version,omitempty"`
	// The peer sync status.
	Synchronized bool `protobuf:"varint,7,opt,name=synchronized,proto3" json:"synchronized,omitempty"`
	// neb version
	Version string `protobuf:"bytes,8,opt,name=version,proto3" json:"version,omitempty"`
}

type ConsensusRoot struct {
	Timestamp   int64  `protobuf:"varint,1,opt,name=timestamp,proto3" json:"timestamp,omitempty,string"`
	Proposer    []byte `protobuf:"bytes,2,opt,name=proposer,proto3" json:"proposer,omitempty"`
	DynastyRoot []byte `protobuf:"bytes,3,opt,name=dynasty_root,json=dynastyRoot,proto3" json:"dynasty_root,omitempty"`
}

type BlockResponse struct {
	// Hex string of block hash.
	Hash string `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	// Hex string of block parent hash.
	ParentHash string `protobuf:"bytes,2,opt,name=parent_hash,json=parentHash,proto3" json:"parent_hash,omitempty"`
	// block height
	Height uint64 `protobuf:"varint,3,opt,name=height,proto3" json:"height,omitempty,string"`
	// block nonce
	Nonce uint64 `protobuf:"varint,4,opt,name=nonce,proto3" json:"nonce,omitempty,string"`
	// Hex string of coinbase address.
	Coinbase string `protobuf:"bytes,5,opt,name=coinbase,proto3" json:"coinbase,omitempty"`
	// block timestamp.
	Timestamp int64 `protobuf:"varint,6,opt,name=timestamp,proto3" json:"timestamp,omitempty,string"`
	// block chain id
	ChainId uint32 `protobuf:"varint,7,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	// Hex string of state root.
	StateRoot string `protobuf:"bytes,8,opt,name=state_root,json=stateRoot,proto3" json:"state_root,omitempty"`
	// Hex string of txs root.
	TxsRoot string `protobuf:"bytes,9,opt,name=txs_root,json=txsRoot,proto3" json:"txs_root,omitempty"`
	// Hex string of event root.
	EventsRoot string `protobuf:"bytes,10,opt,name=events_root,json=eventsRoot,proto3" json:"events_root,omitempty"`
	// Hex string of consensus root.
	ConsensusRoot *ConsensusRoot `protobuf:"bytes,11,opt,name=consensus_root,json=consensusRoot,proto3" json:"consensus_root,omitempty"`
	// Miner
	Miner string `protobuf:"bytes,12,opt,name=miner,proto3" json:"miner,omitempty"`
	// Random seed
	RandomSeed string `protobuf:"bytes,13,opt,name=randomSeed,proto3" json:"randomSeed,omitempty"`
	// Random proof
	RandomProof string `protobuf:"bytes,14,opt,name=randomProof,proto3" json:"randomProof,omitempty"`
	// is finaliy
	IsFinality bool `protobuf:"varint,15,opt,name=is_finality,json=isFinality,proto3" json:"is_finality,omitempty"`
	// transaction slice
	Transactions []*TransactionResponse `protobuf:"bytes,100,rep,name=transactions,proto3" json:"transactions,omitempty"`
}

type TransactionResponse struct {
	// Hex string of tx hash.
	Hash    string `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	ChainId uint32 `protobuf:"varint,2,opt,name=chainId,proto3" json:"chainId,omitempty"`
	// Hex string of the sender account addresss.
	From string `protobuf:"bytes,3,opt,name=from,proto3" json:"from,omitempty"`
	// Hex string of the receiver account addresss.
	To    string `protobuf:"bytes,4,opt,name=to,proto3" json:"to,omitempty"`
	Value string `protobuf:"bytes,5,opt,name=value,proto3" json:"value,omitempty"`
	// Transaction nonce.
	Nonce           uint64 `protobuf:"varint,6,opt,name=nonce,proto3" json:"nonce,omitempty,string"`
	Timestamp       int64  `protobuf:"varint,7,opt,name=timestamp,proto3" json:"timestamp,omitempty,string"`
	Type            string `protobuf:"bytes,8,opt,name=type,proto3" json:"type,omitempty"`
	Data            []byte `protobuf:"bytes,9,opt,name=data,proto3" json:"data,omitempty"`
	GasPrice        string `protobuf:"bytes,10,opt,name=gas_price,json=gasPrice,proto3" json:"gas_price,omitempty"`
	GasLimit        string `protobuf:"bytes,11,opt,name=gas_limit,json=gasLimit,proto3" json:"gas_limit,omitempty"`
	ContractAddress string `protobuf:"bytes,12,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
	// transaction status 0 failed, 1 success, 2 pending
	Status int32 `protobuf:"varint,13,opt,name=status,proto3" json:"status,omitempty"`
	// transaction gas used
	GasUsed string `protobuf:"bytes,14,opt,name=gas_used,json=gasUsed,proto3" json:"gas_used,omitempty"`
	// contract execute error
	ExecuteError string `protobuf:"bytes,15,opt,name=execute_error,json=executeError,proto3" json:"execute_error,omitempty"`
	// contract execute result
	ExecuteResult string `protobuf:"bytes,16,opt,name=execute_result,json=executeResult,proto3" json:"execute_result,omitempty"`
	// transaction's block height
	BlockHeight uint64 `protobuf:"varint,17,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty,string"`
}

type GetAccountStateResponse struct {
	// Current balance in unit of 1/(10^18) nas.
	Balance string `protobuf:"bytes,1,opt,name=balance,proto3" json:"balance,omitempty"`
	// Current transaction count.
	Nonce uint64 `protobuf:"varint,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
	// Account type
	Type uint32 `protobuf:"varint,3,opt,name=type,proto3" json:"type,omitempty"`
	// Block height
	Height uint64 `protobuf:"varint,4,opt,name=height,proto3" json:"height,omitempty"`
	// Current sender pending tx count
	Pending uint64 `protobuf:"varint,5,opt,name=pending,proto3" json:"pending,omitempty"`
}

type GasPriceResponse struct {
	GasPrice string `protobuf:"bytes,1,opt,name=gas_price,json=gasPrice,proto3" json:"gas_price,omitempty"`
}

type PriceResponse struct {
	Result GasPriceResponse `json:"result"`
}

type SendTransactionResponse struct {
	// Hex string of transaction hash.
	Txhash string `protobuf:"bytes,1,opt,name=txhash,proto3" json:"txhash,omitempty"`
	// Hex string of contract address if transaction is deploy type
	ContractAddress string `protobuf:"bytes,2,opt,name=contract_address,json=contractAddress,proto3" json:"contract_address,omitempty"`
}

type CallResponse struct {
	// result of smart contract method call.
	Result string `protobuf:"bytes,1,opt,name=result,proto3" json:"result,omitempty"`
	//execute error
	ExecuteErr string `protobuf:"bytes,2,opt,name=execute_err,json=executeErr,proto3" json:"execute_err,omitempty"`
	//estimate gas used
	EstimateGas string `protobuf:"bytes,3,opt,name=estimate_gas,json=estimateGas,proto3" json:"estimate_gas,omitempty"`
}

type GasResponse struct {
	Gas string `protobuf:"bytes,1,opt,name=gas,proto3" json:"gas,omitempty"`
	Err string `protobuf:"bytes,2,opt,name=err,proto3" json:"err,omitempty"`
}

type CallPayload struct {
	Function string `protobuf:"bytes,3,opt,name=function,proto3" json:"function,omitempty"`
	// the params of contract.
	Args string `protobuf:"bytes,4,opt,name=args,proto3" json:"args,omitempty"`
}

// ToBytes serialize payload
func (payload *CallPayload) ToBytes() ([]byte, error) {
	return json.Marshal(payload)
}

func (payload *CallPayload) Arguments() ([]interface{}, error) {
	if len(payload.Args) > 0 {
		var argsObj []interface{}
		if err := json.Unmarshal([]byte(payload.Args), &argsObj); err != nil {
			return nil, err
		}
		return argsObj, nil
	}
	return []interface{}{}, nil
}

// CheckContractArgs check contract args
func CheckContractArgs(args string) error {
	if len(args) > 0 {
		var argsObj []interface{}
		if err := json.Unmarshal([]byte(args), &argsObj); err != nil {
			return err
		}
	}
	return nil
}

func NewCallPayload(function, args string) (*CallPayload, error) {

	if err := CheckContractArgs(args); err != nil {
		return nil, ErrInvalidArgument
	}

	return &CallPayload{
		Function: function,
		Args:     args,
	}, nil
}

// LoadCallPayload from bytes
func LoadCallPayload(bytes []byte) (*CallPayload, error) {
	payload := &CallPayload{}
	if err := json.Unmarshal(bytes, payload); err != nil {
		return nil, ErrInvalidArgument
	}
	return NewCallPayload(payload.Function, payload.Args)
}
