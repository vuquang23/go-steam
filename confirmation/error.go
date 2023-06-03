package confirmation

import "errors"

var (
	ErrConfirmationsUnknownError = errors.New("unknown error occurered finding confirmations")
	ErrCannotFindConfirmations   = errors.New("unable to find confirmations")
	ErrCannotFindDescriptions    = errors.New("unable to find confirmation descriptions")
	ErrConfirmationsDescMismatch = errors.New("cannot match confirmations with their respective descriptions")
)
