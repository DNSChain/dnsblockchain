package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types" // Descomenta si necesitas tipos de staking
	// authtypes "github.com/cosmos/cosmos-sdk/x/auth/types" // Descomenta si necesitas tipos de auth
)

// AccountKeeper defines the expected interface for the Auth module.
// Esta es la interfaz que tu módulo DAO espera del módulo x/auth.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	// Add other methods from x/auth/keeper.Keeper that your DAO module needs
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error // Firma corregida con context.Context
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	// Add other methods from x/bank/keeper.Keeper that your DAO module needs
}

// DnsblockchainKeeper defines the expected interface for your Dnsblockchain module.
// Añade métodos aquí que el módulo DAO necesita llamar en el DnsblockchainKeeper.
type DnsblockchainKeeper interface {
	AddPermittedTLD(ctx context.Context, tld string) error // Ejemplo que teníamos
	// IsTLDPermitted(ctx context.Context, tld string) bool
}
