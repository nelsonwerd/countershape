//go:build darwin && arm64 && cgo

package hostepoch

/*
#include <errno.h>
#include <stddef.h>
#include <sys/types.h>
#include <sys/sysctl.h>

static int countershape_read_bootsessionuuid(char *buffer, size_t *length) {
	if (sysctlbyname("kern.bootsessionuuid", buffer, length, NULL, 0) != 0) {
		return errno;
	}
	return 0;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func readBootSessionSample() ([]byte, error) {
	var buffer [37]byte
	length := C.size_t(len(buffer))
	code := C.countershape_read_bootsessionuuid((*C.char)(unsafe.Pointer(&buffer[0])), &length)
	if code != 0 {
		return nil, errors.New("fixed Darwin boot-session sysctl failed")
	}
	if int(length) != len(buffer) || buffer[len(buffer)-1] != 0 {
		return nil, errors.New("fixed Darwin boot-session sysctl returned a noncanonical length")
	}
	return append([]byte(nil), buffer[:36]...), nil
}
