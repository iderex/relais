package main

// Deliberately broken, to show the build gate reds on a compilation failure
// rather than being asserted to. It is removed by the next commit on this
// branch, and both runs are linked from the pull request body.
//
// The mistake is a near-miss rather than nonsense: a value returned from a
// function whose signature says it returns none, which is what an edit that
// added the return and forgot the signature leaves behind.
func proofBuildGateBites() {
	return version(nil, false)
}
