package main

import (
	"fmt"
	"io"
)

// This file is deliberate and temporary. It exists for one run, to show that the
// style gate refuses a suppression that names no analyser and gives no reason,
// and that it accepts one that does both. The commit after that run removes it.
//
// Both writes below genuinely trip the unchecked-error analyser, so both
// suppressions suppress something real. A suppression that silences nothing is
// its own finding, and using one here would prove the wrong thing.

// bareSuppression carries the form the convention refuses.
func bareSuppression(w io.Writer) {
	//nolint
	fmt.Fprintln(w, "bare")
}

// namedButUnexplained names its analyser and gives no reason, which is the other
// half of the configuration and reds separately.
func namedButUnexplained(w io.Writer) {
	fmt.Fprintln(w, "named") //nolint:errcheck
}

// formedSuppression carries the form the convention asks for.
func formedSuppression(w io.Writer) {
	fmt.Fprintln(w, "formed") //nolint:errcheck // the destination cannot fail here
}

// Referenced so that the dead-code analyser is not the thing that reds the run.
var _ = []func(io.Writer){bareSuppression, namedButUnexplained, formedSuppression}
