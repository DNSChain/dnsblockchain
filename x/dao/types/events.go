package types

// DAO module event types
const (
	EventTypeSubmitProposal   = "submit_proposal"
	EventTypeProposalVote     = "proposal_vote"
	EventTypeProposalFinished = "proposal_finished"
	EventTypeProposalDeposit  = "proposal_deposit" // <-- NUEVO EVENTO

	AttributeKeyProposalID     = "proposal_id"
	AttributeKeyProposer       = "proposer"
	AttributeKeyVoter          = "voter"
	AttributeKeyVoteOption     = "option"
	AttributeKeyProposalStatus = "proposal_status"
	AttributeKeyVotingPower    = "voting_power"
	AttributeKeyDepositAction  = "deposit_action" // "refunded" o "burned"
	AttributeKeyDepositAmount  = "deposit_amount"
)
