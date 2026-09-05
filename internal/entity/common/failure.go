package common

import "fmt"

type Failure struct {
	Summary string
	Err     error
}

func (rcvr *Failure) Error() string {
	if rcvr == nil {
		return ""
	}
	if rcvr.Err == nil {
		return rcvr.Summary
	}
	return fmt.Sprintf("%s: %v", rcvr.Summary, rcvr.Err)
}

func (rcvr *Failure) Unwrap() error {
	if rcvr == nil {
		return nil
	}
	return rcvr.Err
}
