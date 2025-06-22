package types

// DAO module event types
const (
	EventTypeSubmitProposal   = "submit_proposal"
	EventTypeProposalVote     = "proposal_vote"
	EventTypeProposalFinished = "proposal_finished"
	// Añade más tipos de eventos según se necesiten

	AttributeKeyProposalID     = "proposal_id"
	AttributeKeyProposer       = "proposer"
	AttributeKeyVoter          = "voter"
	AttributeKeyVoteOption     = "option"
	AttributeKeyProposalStatus = "proposal_status"
	// AttributeKeyProposalType ya está definido en govtypes, podemos reusarlo
)
