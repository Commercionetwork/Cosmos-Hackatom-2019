package cli

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/x/pricefeed"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/client/utils"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auth "github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/spf13/cobra"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd(cdc *codec.Codec) *cobra.Command {
	pricefeedTxCmd := &cobra.Command{
		Use:   "pricefeed",
		Short: "Pricefeed transactions subcommands",
	}

	pricefeedTxCmd.AddCommand(client.PostCommands(
		getCmdPostPrice(cdc),
	)...)

	return pricefeedTxCmd
}


// getCmdPostPrice cli command for posting prices.
func getCmdPostPrice(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "postprice [assetCode] [price] [expiry]",
		Short: "post the latest price for a particular asset",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc).WithAccountDecoder(cdc)
			txBldr := auth.NewTxBuilderFromCLI().WithTxEncoder(utils.GetTxEncoder(cdc))
			if err := cliCtx.EnsureAccountExists(); err != nil {
				return err
			}
			price, err := sdk.NewDecFromStr(args[1])
			if err != nil {
				return err
			}
			expiry, ok := sdk.NewIntFromString(args[2])
			if !ok {
				fmt.Printf("invalid expiry - %s \n", string(args[2]))
				return nil
			}
			msg := pricefeed.NewMsgPostPrice(cliCtx.GetFromAddress(), args[0], price, expiry)
			err = msg.ValidateBasic()
			if err != nil {
				return err
			}
			cliCtx.PrintResponse = true
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
}
