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

	timestamp uint64 // used for filter and sorting
}

// IsValid is valid
func (signInfo *SignInfoData) IsValid() bool {
	return signInfo.Key != "" && signInfo.Account != "" && signInfo.GroupID != ""
}

// SignInfoResp sign info response
type SignInfoResp struct {
	Status string
	Tip    string
	Error  string
	Data   []*SignInfoData
}

// SignInfoSortedSlice weighted string slice
type SignInfoSortedSlice []*SignInfoData

// Len impl Sortable
func (s SignInfoSortedSlice) Len() int {
	return len(s)
}

// Swap impl Sortable
func (s SignInfoSortedSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less impl Sortable
func (s SignInfoSortedSlice) Less(i, j int) bool {
	return s[i].timestamp < s[j].timestamp
}

// SignData sign data
type SignData struct {
	TxType     string
	PubKey     string
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
