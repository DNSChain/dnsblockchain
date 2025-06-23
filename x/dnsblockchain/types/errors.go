package types

// DONTCOVER

import (
	"cosmossdk.io/errors"
)

// x/dnsblockchain module sentinel errors
var (
	ErrInvalidSigner       = errors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrDuplicateDomainName = errors.Register(ModuleName, 1101, "domain name already exists")
	ErrInvalidTLD          = errors.Register(ModuleName, 1102, "invalid TLD")
	ErrTLDReservedByICANN  = errors.Register(ModuleName, 1103, "TLD is reserved by ICANN and cannot be registered")
	ErrTLDNotPermitted     = errors.Register(ModuleName, 1104, "the TLD of the domain is not permitted for registration")
	ErrInvalidDomainName   = errors.Register(ModuleName, 1105, "invalid domain name format")
)
