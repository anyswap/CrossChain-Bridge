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

type SignReply struct {
	Enode     string
	Status    string
	TimeStamp string
	Initiator string
}

type SignStatus struct {
	Status    string
	Rsv       string
	Tip       string
	Error     string
	AllReply  []*SignReply
	TimeStamp string
}

type SignInfoData struct {
	Account    string
	GroupID    string
	Key        string
	KeyType    string
	Mode       string
	MsgHash    string
	MsgContext string
	Nonce      string
	PubKey     string
	ThresHold  string
	TimeStamp  string
}

type SignInfoResp struct {
	Status string
	Tip    string
	Error  string
	Data   []*SignInfoData
}

type SignData struct {
	TxType     string
	PubKey     string
	MsgHash    string
	MsgContext string
	Keytype    string
	GroupID    string
	ThresHold  string
	Mode       string
	TimeStamp  string
}

type AcceptData struct {
	TxType    string
	Key       string
	Accept    string
	TimeStamp string
}

type GroupInfo struct {
	Gid    string
	Count  int
	Enodes []string
}

type GetGroupByIDResp struct {
	Status string
	Tip    string
	Error  string
	Data   *GroupInfo
}
