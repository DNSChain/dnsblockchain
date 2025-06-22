// dnsblockchain/x/dao/client/cli/tx.go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	// "github.com/cosmos/cosmos-sdk/codec" // Ya no es necesario aquí
	// codectypes "github.com/cosmos/cosmos-sdk/codec/types" // Ya no es necesario aquí

	"dnsblockchain/x/dao/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(NewSubmitProposalCmd())
	return cmd
}

// NewSubmitProposalCmd returns a CLI command handler for creating a MsgSubmitProposal transaction.
func NewSubmitProposalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-proposal [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a new proposal",
		Long:  "Submit a new proposal supported by the DAO module. Proposal details must be supplied in a JSON file.",
		Example: `dnsblockchaind tx dao submit-proposal path/to/proposal.json --from <key_or_address>
A proposal JSON file example:
{
  "content": {
    "@type": "/dnsblockchain.dao.v1.AddTldProposalContent",
    "tld": "web3",
    "description": "Propuesta para añadir el TLD .web3"
  },
  "title": "Añadir TLD .web3",
  "description": "Esta propuesta busca añadir el dominio de nivel superior .web3 a la lista de TLDs permitidos en la plataforma.",
  "proposer": "cosmos1...", // Proposer address
  "initial_deposit": [
    {
      "denom": "stake",
      "amount": "10000000"
    }
  ]
}
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposalFilePath := args[0]

			contents, err := os.ReadFile(proposalFilePath)
			if err != nil {
				return fmt.Errorf("failed to read proposal file: %w", err)
			}

			var msg types.MsgSubmitProposal
			// Usar clientCtx.Codec directamente. Este debería ser el ProtoCodec de la aplicación.
			if err := clientCtx.Codec.UnmarshalJSON(contents, &msg); err != nil {
				return fmt.Errorf("failed to unmarshal proposal JSON: %w (ensure @type for content is correct and registered)", err)
			}

			fromAddr := clientCtx.GetFromAddress()
			if msg.Proposer != "" && msg.Proposer != fromAddr.String() {
				return fmt.Errorf("proposer in JSON file (%s) does not match the --from address (%s)", msg.Proposer, fromAddr.String())
			}
			if msg.Proposer == "" {
				msg.Proposer = fromAddr.String()
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
