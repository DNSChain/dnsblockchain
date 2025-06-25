package keeper

import (
	"dnsblockchain/x/dnsblockchain/types"
)

// queryServer es la implementación de types.QueryServer
type queryServer struct {
	k Keeper // Contiene la instancia del Keeper principal
}

var _ types.QueryServer = queryServer{} // Asegura la conformidad con la interfaz

// NewQueryServerImpl crea una nueva instancia de QueryServer.
func NewQueryServerImpl(k Keeper) types.QueryServer {
	return queryServer{k: k}
}

// Las implementaciones de los métodos individuales (Params, GetDomain, ListDomain, GetDomainByName, ListPermittedTLDs)
// estarán en sus archivos query_*.go como métodos de la struct queryServer.
