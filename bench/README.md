# bench

The measuring instrument, outside the build of the service on purpose. Nothing
under `cmd/` or `internal/` may import anything here, and the bench may not
import the service it measures, because a bench that has to be linked against
this project cannot be pointed at anything else and a bench that only fits its
author's software proves nothing. It measures a named target, records the
machine, the kernel, the version, the offered load and the command that produced
every figure, and it writes a result file somebody on different hardware can
repeat. It is built before the thing it measures, for the same reason an
instrument is calibrated before the experiment: built afterwards, it gets built
to produce a flattering number. Issue #8 is where it is made.
