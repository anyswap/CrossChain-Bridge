/*
Package data aims to provides all the data types that are needed to build tools,
clients and servers for use on the Ripple network.

Ledger

The ledger is a mixture of various data types, all persisted in the form of a
key and value. The value contains a node, and some nodes refer to other nodes by
their index. An index may or may not be equivalent to a node's key. A ledger
consists of a LedgerHeader as the root of two trees, one for the transactions in
that ledger and one for the state of all accounts at that point in time.

A ledger refers to its predecessor, which has a completely different tree for
its transactions, but the account state tree will share most of its nodes with
its successors and predecessors.

The full ledger consists of a sequence of such ledgers, each referring to, and
dependent on its predecessor.

Important terms are key, value, index, hash and node. Keys, indexes and hashes
are always 32 bytes in length and involve the use of the SHA512 cryptographic
hash algorithm, discarding the last 32 bytes of the resulting hash. Sometimes
they are the same thing, sometimes not.

LedgerHeader

This is the root of the two trees for a single ledger. It contains information
about the ledger and its predecessor. A ledger header node is never reused
between ledgers.

	Node:	Simple big-endian binary encoding of LedgerHeader
	Index:  SHA512Half of HP_LEDGER_MASTER:Node
	Hash:	Same as Index

	Key:	Index
	Value:	LedgerSequence:LedgerSequence:NT_LEDGER:HP_LEDGER_MASTER:Node

Inner Node

This is a node in either the transaction or account state tree which contains up
to 16 hashes of other nodes. The position of the hash represents a 4 bit nibble
of the index that is being searched for in the tree. An account state inner node
may be reused between ledgers, a transaction inner node will never be reused.

	Node:	Simple big-endian binary encoding of 16 x 32 byte hashes
	Index:	SHA512Half of HP_INNER_NODE:Node
	Hash:	Same as Index

	Key: 	Index
	Value:	LedgerSequence:LedgerSequence:NT_ACCOUNT_NODE or NT_TRANSACTION_NODE:HP_INNER_NODE:Node

TransactionWithMetadata Node

This contains a Transaction with Metadata describing how the LedgerEntry nodes
were altered. It has the most complex structure of all nodes. A
TransactionWithMetadata node is never reused between ledgers. VL is a variable
length marker.

	Node: 	Complex encoding of Transaction
	Index:	SHA512Half of HP_TRANSACTION_ID:Node
	Hash:	SHA512Half of HP_TRANSACTION_NODE:VL:Node:VL:Metadata:Index

	Key:	Hash
	Value:	LedgerSequence:LedgerSequence:NT_TRANSACTION_NODE:HP_TRANSACTION_NODE:VL:Node:VL:Metadata:Index

LedgerEntry Node

This contains the state of an account after a transaction has been applied.
Ledger Entry nodes may be reused between ledgers.

	Node:	Complex encoding of LedgerEntry
	Index:	SHA512Half of namespace and a type-specific rule
	Hash:	SHA512Half of HP_LEAF_NODE:Node:Index

	Key:	Hash
	Value:	LedgerSequence:LedgerSequence:NT_ACCOUNT_NODE:HP_LEAF_NODE:Node:Index

*/
package data
