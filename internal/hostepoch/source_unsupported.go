//go:build !darwin || !arm64 || !cgo

package hostepoch

func readBootSessionSample() ([]byte, error) {
	return nil, refuse(CodeUnsupportedPlatform, "DARWIN_KERN_BOOTSESSIONUUID_V1 requires Darwin arm64 with cgo", nil)
}
