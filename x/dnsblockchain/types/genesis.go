package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{ // Esto fallará si el tipo GenesisState del .pb.go no tiene PermittedTlds
		Params:        DefaultParams(),
		DomainList:    []Domain{},
		PermittedTlds: []string{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	domainIdMap := make(map[uint64]bool)
	domainCount := gs.GetDomainCount() // Método generado por .pb.go
	for _, elem := range gs.DomainList {
		if _, ok := domainIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for domain")
		}
		if elem.Id >= domainCount {
			return fmt.Errorf("domain id should be lower or equal than the last id")
		}
		domainIdMap[elem.Id] = true
	}

	permittedTLDsMap := make(map[string]bool)
	for _, tld := range gs.GetPermittedTlds() { // Usar el getter generado por .pb.go
		if permittedTLDsMap[tld] {
			return fmt.Errorf("duplicated permitted TLD: %s", tld)
		}
		permittedTLDsMap[tld] = true
	}

	return gs.Params.Validate()
}
