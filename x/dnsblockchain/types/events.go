package types

// Events for the dnsblockchain module
const (
	EventTypeCreateDomain       = "create_domain"
	EventTypeUpdateDomain       = "update_domain"
	EventTypeDeleteDomain       = "delete_domain"
	EventTypeTransferDomain     = "transfer_domain"
	EventTypeHeartbeatDomain    = "heartbeat_domain"
	EventTypeDomainFeeCollected = "domain_fee_collected" // Para la tarifa de creación
	EventTypeDomainFeeBurned    = "domain_fee_burned"    // Para la quema de la tarifa

	AttributeKeyDomainID      = "domain_id"
	AttributeKeyDomainName    = "domain_name"
	AttributeKeyOwner         = "owner"
	AttributeKeyCreator       = "creator" // Usado en CreateDomain
	AttributeKeyUpdater       = "updater" // Usado en UpdateDomain
	AttributeKeyActor         = "actor"   // Usado en Delete, Heartbeat, Transfer (quien realiza la acción)
	AttributeKeyNewOwner      = "new_owner"
	AttributeKeyOldOwner      = "old_owner"
	AttributeKeyNewExpiration = "new_expiration"
	AttributeKeyFeeCollector  = "fee_collector" // Quién pagó la tarifa
	AttributeKeyBurnerModule  = "burner_module" // Módulo que quemó la tarifa
	// sdk.AttributeKeyAmount se puede usar para el monto de la tarifa
)
