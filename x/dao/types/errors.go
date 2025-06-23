package types

import (
	"cosmossdk.io/errors"
)

// x/dao module sentinel errors
var (
	ErrInvalidProposalContent  = errors.Register(ModuleName, 1203, "invalid proposal content")
	ErrProposalNotFound        = errors.Register(ModuleName, 1204, "proposal not found")
	ErrInactiveProposal        = errors.Register(ModuleName, 1205, "proposal not in voting period")
	ErrVotingPeriodEnded       = errors.Register(ModuleName, 1206, "voting period has ended")
	ErrAlreadyVoted            = errors.Register(ModuleName, 1207, "voter has already voted on this proposal")
	ErrNoVotingPower           = errors.Register(ModuleName, 1208, "voter has no voting power")
	ErrDuplicateActiveProposal = errors.Register(ModuleName, 1209, "an identical proposal is already in its voting period")
	// Añade ErrEmptyDescription y ErrNonPositiveAmount si los necesitas aquí
	// o usa los de sdkerrors.ErrInvalidRequest envueltos.
)
