package types

// DAO module event types
const (
	EventTypeSubmitProposal = "submit_proposal"
	EventTypeProposalVote   = "proposal_vote"
	// Añade más tipos de eventos según se necesiten

	AttributeKeyProposalID = "proposal_id"
	AttributeKeyProposer   = "proposer"
	AttributeKeyVoter      = "voter"
	AttributeKeyVoteOption = "option"
	// AttributeKeyProposalType ya está definido en govtypes, podemos reusarlo
)
