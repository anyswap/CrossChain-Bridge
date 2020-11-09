package dcrm

// DataEnode enode
type DataEnode struct {
	Enode string
}

// GetEnodeResp enode response
type GetEnodeResp struct {
	Status string
	Tip    string
	Error  string
	Data   *DataEnode
}

// DataResult result
type DataResult struct {
	Result string `json:"result"`
}

// DataResultResp result response
type DataResultResp struct {
	Status string
	Tip    string
	Error  string
	Data   *DataResult
}

// SignReply sign reply
type SignReply struct {
	Enode     string
	Status    string
	TimeStamp string
	Initiator string
}

// SignStatus sign status
type SignStatus struct {
	Status    string
	Rsv       []string
	Tip       string
	Error     string
	AllReply  []*SignReply
	TimeStamp string
}

// SignInfoData sign info
type SignInfoData struct {
	Account    string
	GroupID    string
	Key        string
	KeyType    string
	Mode       string
	MsgHash    []string
	MsgContext []string
	Nonce      string
	PubKey     string
	ThresHold  string
	TimeStamp  string
}

// SignInfoResp sign info response
type SignInfoResp struct {
	Status string
	Tip    string
	Error  string
	Data   []*SignInfoData
}

// SignData sign data
type SignData struct {
	TxType     string
	PubKey     string
	InputCode  string
	MsgHash    []string
	MsgContext []string
	Keytype    string
	GroupID    string
	ThresHold  string
	Mode       string
	TimeStamp  string
}

// AcceptData accpet data
type AcceptData struct {
	TxType     string
	Key        string
	Accept     string
	MsgHash    []string
	MsgContext []string
	TimeStamp  string
}

// GroupInfo group info
type GroupInfo struct {
	GID    string
	Count  int
	Enodes []string
}

// GetGroupByIDResp group response
type GetGroupByIDResp struct {
	Status string
	Tip    string
	Error  string
	Data   *GroupInfo
}
