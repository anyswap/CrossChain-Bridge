// Package worker includes all the tasks and jobs to process bridge swaps.
//
// It contains the following main steps (concurrently):
//	verify
//		verify registered swaps.
//	swap
//		build swaptx, mpc sign the tx, and send the tx to blockchain.
//	accept
//		the `oracle` node do the accept job, agree or disagree the signing after verifying by oralce itself.
//	stable
//		mark swap status to `stabe` status.
//	replace
//		replace swap with the same tx nonce value when the sent swaptx is not packed into block because of lack fee or other reasons.
//	passbigvalue
//		pass big value swap if the swap value is too large.
// Most the above jobs is assigned to the `server` node, the `oracle` node mainly do the `accept` job.
package worker
