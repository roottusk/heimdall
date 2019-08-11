package tx

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/spf13/cobra"
	amino "github.com/tendermint/go-amino"

	authTypes "github.com/maticnetwork/heimdall/auth/types"
	"github.com/maticnetwork/heimdall/helper"
)

type (
	// EncodeReq defines a tx encoding request.
	EncodeReq struct {
		Tx authTypes.StdTx `json:"tx"`
	}

	// EncodeResp defines a tx encoding response.
	EncodeResp struct {
		Tx string `json:"tx"`
	}
)

// EncodeTxRequestHandlerFn returns the encode tx REST handler. In particular,
// it takes a json-formatted transaction, encodes it to the Amino wire protocol,
// and responds with base64-encoded bytes.
func EncodeTxRequestHandlerFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req EncodeReq

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		err = cdc.UnmarshalJSON(body, &req)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// re-encode it via the Amino wire protocol
		txBytes, err := cliCtx.Codec.MarshalBinaryLengthPrefixed(req.Tx)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		// base64 encode the encoded tx bytes
		txBytesBase64 := base64.StdEncoding.EncodeToString(txBytes)

		response := EncodeResp{Tx: txBytesBase64}
		rest.PostProcessResponse(w, cdc, response, cliCtx.Indent)
	}
}

// GetEncodeCommand returns the encode command to take a JSONified transaction and turn it into
// Amino-serialized bytes
func GetEncodeCommand(codec *amino.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encode [file]",
		Short: "Encode transactions generated offline",
		Long: `Encode transactions created with the --generate-only flag and signed with the sign command.
Read a transaction from <file>, serialize it to hex. 
If you supply a dash (-) argument in place of an input filename, the command reads from standard input.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cliCtx := context.NewCLIContext().WithCodec(codec)

			stdTx, err := helper.ReadStdTxFromFile(cliCtx.Codec, args[0])
			if err != nil {
				return
			}

			// re-encode it via the Amino wire protocol
			txBytes, err := helper.GetStdTxBytes(cliCtx, stdTx)
			response := hex.EncodeToString(txBytes)
			fmt.Println("Tx:", response)

			return nil
		},
	}

	return client.PostCommands(cmd)[0]
}
