package dcrm

type DataEnode struct {
	Enode string
}

type GetEnodeResp struct {
	Status string
	Tip    string
	Error  string
	Data   *DataEnode
}

type DataResult struct {
	Result string `json:"result"`
}

type DataResultResp struct {
	Status string
	Tip    string
	Error  string
	Data   *DataResult
}

type SignStatus struct {
	Rsv       string
	AllReply  interface{}
	TimeStamp string
}

type SignStatusResp struct {
	Status    string
	Rsv       string
	Tip       string
	Error     string
	AllReply  interface{}
	TimeStamp string
}

type SignInfoData struct {
	Account   string
	GroupID   string
	Key       string
	KeyType   string
	Mode      string
	MsgHash   string
	Nonce     string
	PubKey    string
	ThresHold string
	TimeStamp string
}

type SignInfoResp struct {
	Status string
	Tip    string
	Error  string
	Data   []*SignInfoData
}

type SignData struct {
	TxType    string
	PubKey    string
	MsgHash   string
	Keytype   string
	GroupID   string
	ThresHold string
	Mode      string
	TimeStamp string
}

type AcceptData struct {
	TxType    string
	Key       string
	Accept    string
	TimeStamp string
}
