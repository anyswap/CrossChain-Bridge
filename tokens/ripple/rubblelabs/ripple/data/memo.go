package data

type Memo struct {
	Memo struct {
		MemoType   VariableLength
		MemoData   VariableLength
		MemoFormat VariableLength
	}
}

type Memos []Memo
