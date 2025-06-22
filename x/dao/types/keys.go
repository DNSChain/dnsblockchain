package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "dao"

	// StoreKey defines the primary module store key
	// Este StoreKey se usa al registrar el módulo en app.go
	// y para obtener el KVStoreService para el keeper.
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key (si lo necesitas, para queries temporales, etc.)
	// MemStoreKey = "mem_dao"
)

// Prefijos para las claves del store usadas con cosmossdk.io/collections
var (
	// ParamsKey es el prefijo para los parámetros del módulo.
	ParamsKey = collections.NewPrefix("Params/") // Es buena práctica terminar los prefijos con '/'

	// ProposalSeqKey es la clave para la secuencia de IDs de propuestas.
	ProposalSeqKey = collections.NewPrefix("ProposalSeq/")

	// ProposalsKeyPrefix es el prefijo para almacenar propuestas por su ID.
	ProposalsKeyPrefix = collections.NewPrefix("Proposals/value/")

	// VotesKeyPrefix es el prefijo para almacenar votos.
	// La clave completa será algo como "Votes/value/ProposalID/VoterAddress".
	VotesKeyPrefix = collections.NewPrefix("Votes/value/")

	// ActiveProposalQueueKeyPrefix es el prefijo para la cola de propuestas activas.
	// La implementación exacta de la cola podría variar (ej. (EndTime, ProposalID) -> nil).
	ActiveProposalQueueKeyPrefix = collections.NewPrefix("ActiveProposalQueue/")
)

// Función para generar la clave de una propuesta específica (opcional, pero útil)
// func GetProposalKey(proposalID uint64) []byte {
// 	return append(ProposalsKeyPrefix, sdk.Uint64ToBigEndian(proposalID)...)
// }

// Función para generar la clave de un voto específico (opcional, pero útil)
// func GetVoteKey(proposalID uint64, voter sdk.AccAddress) []byte {
// 	key := append(VotesKeyPrefix, sdk.Uint64ToBigEndian(proposalID)...)
// 	key = append(key, 길이_접두사_주소(voter)...) // Address.Bytes() con prefijo de longitud si es necesario
// 	return key
// }

// Nota: Con `collections`, la construcción de claves complejas como para Votes
// (con PairKey) se maneja internamente por la librería al usar .Get() o .Set(),
// por lo que estas funciones helper para generar claves de bytes completas
// son menos necesarias que con el API de store anterior.
// Lo importante son los prefijos para `NewMap`, `NewSequence`, etc.
