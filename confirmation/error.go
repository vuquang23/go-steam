package confirmation

import "errors"

var (
	ErrCannotFindOffer        = errors.New("unable to find offer")
	ErrCannotGetConfirmations = errors.New("unable to get confirmations")
)
