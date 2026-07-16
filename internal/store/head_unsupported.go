//go:build !darwin

package store

import (
	"context"
	"errors"
)

type studyLock struct{}

func openAndLockStudy(context.Context, string, bool) (studyLock, error) {
	return studyLock{}, errors.New("study-head CAS is UNRECEIPTED and unavailable outside Darwin")
}
func (studyLock) release() {}
func renameHeadNoFollow(string, string) error {
	return errors.New("study-head CAS is UNRECEIPTED and unavailable outside Darwin")
}
