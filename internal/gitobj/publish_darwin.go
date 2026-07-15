//go:build darwin && cgo

package gitobj

/*
#include <errno.h>
#include <stdlib.h>
#include <sys/stdio.h>

static int countershape_rename_exclusive(const char *from, const char *to, int exclusive) {
	unsigned int flags = RENAME_NOFOLLOW_ANY;
	if (exclusive) flags |= RENAME_EXCL;
	if (renamex_np(from, to, flags) == 0) return 0;
	return errno;
}
*/
import "C"

import (
	"syscall"
	"unsafe"
)

const exclusivePublicationRequired = true // MUTANT_U2_ALLOW_DESTINATION_OVERWRITE

func renameExclusive(from, to string) error {
	fromCString := C.CString(from)
	toCString := C.CString(to)
	defer C.free(unsafe.Pointer(fromCString))
	defer C.free(unsafe.Pointer(toCString))
	exclusive := C.int(0)
	if exclusivePublicationRequired {
		exclusive = 1
	}
	if errorNumber := C.countershape_rename_exclusive(fromCString, toCString, exclusive); errorNumber != 0 {
		return syscall.Errno(errorNumber)
	}
	return nil
}
