#TODO

##Data
* Write good tests for metadata interpretation
* Use Freeform type for _some_ memos and Previous/New/Final fields
* Implement canonical signatures
* Consider adding SuppressionId, NodeId, SigningHash and Hash to hashable interface and make the encoder do all four in one pass. Raw is the full encoded value with every field included.

##Peers
* Implement all handlers

##Ledger
* Allow subscribing to incoming Proposals/Validations/Transactions for use in listener


##Terminal

##Websockets
* Add missing commands
* Make connection resilient via reconnection strategy (r.ripple.com?)
* Allow connection to multiple endpoints?

##Tools

###tx
* Implement OfferCreate, OfferCancel, AccountSet and TrustSet commands
* Use websockets to optionally acquire correct sequence number for account derived from seed 
* Add memo support
