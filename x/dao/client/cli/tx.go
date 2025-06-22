// dnsblockchain/x/dao/client/cli/tx.go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	// Usaremos esto para parsear la propuesta
	// govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1" // O v1 si es necesario para el parsing

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
	// cmd.AddCommand(NewVoteCmd()) // Podrías mover Vote aquí también si Autocli da problemas

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
			// Parseamos el archivo JSON usando una función similar a la de x/gov
			// o implementamos una nosotros mismos.
			// Por simplicidad, vamos a intentar parsear directamente MsgSubmitProposal.
			// Si esto falla por el campo Any, podríamos necesitar un struct intermedio.

			contents, err := os.ReadFile(proposalFilePath)
			if err != nil {
				return fmt.Errorf("failed to read proposal file: %w", err)
			}

			var msg types.MsgSubmitProposal
			// Necesitamos usar el codec de la interfaz para deserializar Any correctamente.
			// clientCtx.InterfaceRegistry debería estar disponible.
			// O usar ModuleCdc directamente si está configurado con la InterfaceRegistry.
			if err := types.ModuleCdc.UnmarshalJSON(contents, &msg); err != nil {
				// Intenta también con el codec del clientCtx si el ModuleCdc no funciona para Any
				if err := clientCtx.Codec.UnmarshalJSON(contents, &msg); err != nil {
					return fmt.Errorf("failed to unmarshal proposal: %w (ensure @type is correct for content)", err)
				}
			}

			// El campo 'proposer' en el JSON se puede usar para validación,
			// pero el firmante real vendrá de --from.
			// Para asegurar consistencia, podemos establecer el msg.Proposer al firmante.
			// O validar que coincidan.
			fromAddr := clientCtx.GetFromAddress()
			if msg.Proposer != "" && msg.Proposer != fromAddr.String() {
				return fmt.Errorf("proposer in JSON file (%s) does not match the --from address (%s)", msg.Proposer, fromAddr.String())
			}
			// Si el proposer no está en el JSON, lo asignamos desde el flag --from
			if msg.Proposer == "" {
				msg.Proposer = fromAddr.String()
			}

			// Validar el contenido del mensaje aquí si es necesario, aunque ValidateBasic se llamará después
			// if err := msg.ValidateBasic(); err != nil {
			// 	return err
			// }

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	// No necesitamos flags específicos para los campos de MsgSubmitProposal
	// porque todos vienen del archivo JSON.

	return cmd
}
