package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/dnsblockchain module sentinel errors
var (
	ErrInvalidSigner       = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrDuplicateDomainName = errors.Register(ModuleName, 1101, "domain name already exists")
	ErrInvalidTLD          = errors.Register(ModuleName, 1102, "invalid TLD") // <-- NUEVO ERROR
)
