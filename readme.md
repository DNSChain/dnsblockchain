# DNS Chain ⛓️🌐

**DNS Chain** is a sovereign blockchain built with **Cosmos SDK** and **CometBFT**, designed to revolutionize the management and ownership of Top-Level Domains (TLDs) and domains through a decentralized autonomous organization (DAO). This project was initiated with [Ignite CLI](https://ignite.com/cli).

We believe in a more open, transparent, and user-controlled internet. DNS Chain empowers its community to propose, vote on, and manage the creation of new TLDs, as well as participate in the evolution of the platform itself.

**Join our DAO and help us build the future of decentralized DNS!**

## ✨ Project Vision

The current DNS system, while fundamental, is centralized and susceptible to censorship and single points of failure. DNS Chain aims to:

*   **Decentralize TLD Creation:** Allow the community, through the DAO, to propose and approve new TLDs, fostering innovation and diversity in the internet namespace.
*   **Verifiable Domain Ownership:** Register ownership of `label.TLD` domains on the blockchain, offering transparency and censorship resistance.
*   **Community Governance:** Ensure all major platform decisions, including TLD additions, system parameters, and upgrades, are made by the holders of the governance token (`udns`).
*   **Interoperability:** Leverage the Cosmos ecosystem for future integrations and inter-chain communication (IBC).

## 🚀 Key Features

*   **DNS Blockchain Module:** Enables the registration and management of `label.TLD` domains under DAO-approved TLDs. Includes functionalities like domain creation, updates, transfers, and renewals (heartbeats).
*   **DAO (Decentralized Autonomous Organization) Module:**
    *   Proposal system for adding new TLDs and requesting community funds.
    *   Voting mechanism based on the `udns` token, with voting power for granted tokens that can decay over time.
    *   DAO-configurable parameters (quorum, voting thresholds, proposal costs, voting power decay duration).
    *   Deny list of ICANN-reserved TLDs to prevent conflicts with the global DNS.
*   **Built on Cosmos SDK:** Inherits the modularity, security, and performance of the Cosmos framework.
*   **CometBFT Consensus:** Utilizes CometBFT (formerly Tendermint Core) for fast and secure BFT consensus.

## 🤝 How to Contribute & Join the DAO!

We are looking for passionate contributors interested in decentralization, DNS, and community governance! There are many ways to get involved:

1.  **Code Development:**
    *   Check out our [Issues on GitHub](https://github.com/YOUR_USERNAME/dnsblockchain/issues) (replace with your actual link!).
    *   Propose new features or improvements.
    *   Help with refactoring, optimization, and writing tests.
    *   Contribute to technical documentation.

2.  **DAO Participation:**
    *   **Acquire `udns` tokens:** The primary way to obtain `udns` with specialized voting power will be through DAO-approved `RequestTokensProposalContent` proposals recognizing contributions. The `udns` token is also the native token for staking and gas.
    *   **Create Proposals:** Once you have `udns` (and meet deposit requirements), you can propose new TLDs, request funds for projects benefiting the DNS Chain ecosystem, or propose changes to DAO or chain parameters.
    *   **Vote on Proposals:** Actively participate in decision-making by voting on existing proposals.
    *   **Debate and Discussion:** Join our communication channels (see below) to discuss ideas, proposals, and the future of DNS Chain.

3.  **Testing and Feedback:**
    *   Run a local node (see "Get Started (Local Development)" below).
    *   Test existing functionalities and report bugs or suggest improvements.

4.  **Documentation and Community:**
    *   Help improve our user and developer documentation.
    *   Engage with the community, help onboard new members, and spread the word about DNS Chain.

## 🏁 Get Started (Local Development)

To start developing or testing DNS Chain locally:

# Clone the repository (replace with your URL)
git clone https://github.com/YOUR_USERNAME/dnsblockchain.git
cd dnsblockchain

# Install dependencies, build, initialize, and start the development chain
ignite chain serve --reset-once

The serve command will provide you with:
- Test account addresses (Alice, Bob) and their mnemonics.
- Endpoints for the CometBFT node (RPC) and the blockchain API (REST/gRPC).
- A token faucet (if configured in config.yml).

🛠️ Configuration
Your development blockchain can be configured with config.yml. Learn more in the Ignite CLI documentation.

The primary native and governance token is udns.

🌐 Web Frontend (Optional)

Ignite CLI offers options for scaffolding a frontend:
- Vue: ignite scaffold vue
- React: ignite scaffold react

🏛️ DAO Governance in Detail
- The DAO module is at the heart of decision-making on DNS Chain.
- Voting Token: udns
-- Current Proposal Types:
-- AddTldProposalContent: To propose adding a new TLD to the permitted list.
- RequestTokensProposalContent: To request funds (in udns or other tokens) from the DAO module's treasury (if a treasury is implemented) or to mint new udns tokens that grant voting power.

- Voting Power:
-- Initially (before the first RequestTokens proposal is executed), voting power is based on an account's general udns balance.
-- Once the first RequestTokens proposal is executed, voting power is primarily derived from "lots" of udns granted by such proposals.
-- These voting power lots decay linearly over time (configured by VotingPowerDecayDurationBlocks, defaults to 100 blocks for testing, but configurable to ~6 months for production).
- Key Parameters (configurable via DAO governance or genesis):
-- VotingPeriodBlocks: Duration of the voting period.
-- ProposalSubmissionDeposit: Deposit for general proposals.
-- AddTldProposalCost: Specific cost for TLD proposals.
-- QuorumPercent: Minimum percentage of total voting power (at snapshot) that must participate.
-- MinYesThresholdPercent: Minimum percentage of YES votes (over YES+NO votes) to pass.

(Release)
To release a new version of your blockchain, create and push a new tag with a v prefix. A new draft release with the configured targets will be created.

git tag v0.1.0

git push origin v0.1.0

💬 Join the Conversation
- GitHub Discussions: https://github.com/DNSChain/dnsblockchain/discussions
- Bitcoin Talk: https://bitcointalk.org/index.php?topic=5547781.0