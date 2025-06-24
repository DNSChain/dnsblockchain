**Decentralized DNS Infrastructure on Blockchain: A Whitepaper**

**Version:** 1.0  
**Date:** June 2025  
**Authors:** Community-driven Project (Open Specification)

---

## Abstract

The Domain Name System (DNS), a critical infrastructure of the modern Internet, remains centralized and monopolized by a handful of entities. This whitepaper proposes a decentralized, blockchain-based DNS infrastructure that enables open registration and delegation of domain names using a Proof-of-Stake (PoS) blockchain. By limiting on-chain data to authoritative name servers (NS) only, the system remains efficient, scalable, and interoperable with traditional DNS infrastructure. The network incentivizes participation through staking and fee-sharing mechanisms and preserves compatibility with the ICANN-based DNS system.

---

## 1. Introduction

DNS has long been managed by centralized authorities such as ICANN and commercial registrars. These entities control registration policies and impose arbitrary costs and restrictions, leading to censorship risks, domain squatting, and high registration fees.

Our proposed decentralized DNS protocol addresses these issues by:
- Allowing domain registration on a permissionless blockchain.
- Delegating domain resolution to user-specified name servers.
- Maintaining backward compatibility with ICANN domains.
- Incentivizing decentralized participation via Proof-of-Stake.

---

## 2. System Overview

The protocol consists of the following components:

- **Blockchain Ledger:** Maintains records of domain ownership and associated name servers (NS records only).
- **PoS Consensus Layer:** Secures the blockchain and rewards validators with transaction fees.
- **DNS Client (Resolver):** Software acting as a hybrid resolver, supporting both decentralized and ICANN DNS resolution.
- **Domain Owner Wallets:** Allow users to register, update, or transfer domains via private key signatures.

---

## 3. Domain Registration Protocol

### 3.1 On-Chain Data
Only authoritative name servers (NS) are stored on-chain. Other DNS records (A, AAAA, CNAME, MX, TXT) are served by the off-chain authoritative servers designated in the NS records.

### 3.2 Registration Process
- Domain owner sends a transaction specifying:
  - Domain name (max length enforced)
  - Up to 6 NS records
  - Network fee paid in the native token
- Validators confirm the transaction, update the domain registry.
- Domain is considered active upon block confirmation.

### 3.3 Heartbeat & Expiry
- Domains require an annual "heartbeat" transaction signed by the owner.
- If no heartbeat is detected within 365 days, the domain expires and becomes available for re-registration.

### 3.4 Transfers
- Domains may be transferred between wallets through a signed transaction, with a fee paid to the network.

---

## 4. TLD Management

To maintain compatibility with the existing DNS infrastructure, a hardcoded list of ICANN TLDs (e.g., `.com`, `.net`, `.org`, country-code TLDs, etc.) is excluded from registration on the blockchain.

The protocol only permits registration of domains with non-conflicting TLDs (e.g., `.dweb`, `.alt`, `.peer`, etc.), ensuring no collision with ICANN-rooted DNS.

---

## 5. Hybrid DNS Resolution

The client DNS resolver implements the following logic:

1. Extract the TLD of the queried domain.
2. If the TLD is in the hardcoded exclusion list:
   - Resolve via traditional ICANN DNS resolvers (e.g., 8.8.8.8).
3. Otherwise:
   - Query the blockchain for NS records.
   - If found, resolve using the designated authoritative servers.
   - If not found, return NXDOMAIN or fallback to ICANN.

---

## 6. Incentives and Governance

### 6.1 Staking Requirements
- Participants must stake tokens and run an active node for a minimum of 30 days before becoming eligible to receive network fees.

### 6.2 Fee Distribution
- All domain registration, update, and transfer fees are distributed proportionally to staked validators.

### 6.3 Governance
- Future protocol updates and TLD admission policies may be governed through on-chain voting by token holders.

---

## 7. Advantages Over Existing Systems (Expanded)

| Feature                    | DNSChain (Decentralized DNS) | ENS   | Unstoppable Domains | ICANN DNS | Handshake | Namecoin | EmerDNS | GNUnet (GNS) | OpenNIC |
|---------------------------|------------------------------|-------|----------------------|------------|-----------|----------|---------|---------------|---------|
| **Rootless**              | Yes                          | Partially | No                | No         | Yes       | Yes      | Yes     | Yes           | No      |
| **TLD Flexibility**       | High                         | Medium | Low (predefined)    | Low        | High      | Low      | Medium  | Medium        | High    |
| **Interoperable**         | Yes                          | Limited to Ethereum | No         | Yes        | Limited   | No       | No      | No            | Yes     |
| **Staking Economy**       | Yes                          | No    | No                  | No         | No        | No       | No      | No            | No      |
| **Censorship Resistance** | High                         | Medium | Medium              | Low        | High      | High     | Medium  | High          | Medium  |

---

## 8. Roadmap (Tentative)

- **Phase 1:** Design and testnet deployment of the blockchain and DNS resolver client.
- **Phase 2:** Launch of public mainnet with validator onboarding and domain registration.
- **Phase 3:** Community governance and ecosystem tooling (explorer, dashboards).
- **Phase 4:** Interoperability bridges, registrar APIs, and mobile DNS clients.

---

## 9. Conclusion

This project proposes a practical and interoperable approach to decentralizing DNS. By leveraging a lightweight blockchain design that stores only NS records and supports resolution fallback to ICANN, it ensures immediate usability, censorship resistance, and openness to innovation. Community participation, staking incentives, and open governance make this protocol sustainable and scalable for global adoption.

We invite developers, DNS operators, and decentralized infrastructure advocates to collaborate in shaping the future of DNS.

---

## 10. References
- ICANN Root Zone Database
- Handshake Whitepaper
- Ethereum Name Service Docs
- DNS RFCs (RFC 1034, RFC 1035)
- DNS over HTTPS / TLS standards
