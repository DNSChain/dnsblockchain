package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		DomainList: []Domain{}}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	domainIdMap := make(map[uint64]bool)
	domainCount := gs.GetDomainCount()
	for _, elem := range gs.DomainList {
		if _, ok := domainIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for domain")
		}
		if elem.Id >= domainCount {
			return fmt.Errorf("domain id should be lower or equal than the last id")
		}
		domainIdMap[elem.Id] = true
	}

	return gs.Params.Validate()
}
