//go:build !darwin

package scope

func Snapshot(string) (Inventory, error) {
	return Inventory{}, fail(CodeInventoryInvalid, nil)
}
