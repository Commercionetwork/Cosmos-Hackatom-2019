package rest

import (
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/gorilla/mux"

	"github.com/kava-labs/kava-devnet/blockchain/x/auction"

	clientrest "github.com/cosmos/cosmos-sdk/client/rest"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type placeBidReq struct {
	BaseReq   rest.BaseReq `json:"base_req"`
	AuctionID string       `json:"auction_id"`
	Bidder    string       `json:"bidder"`
	Bid       string       `json:"bid"`
	Lot       string       `json:"lot"`
}

const (
	restAuctionID = "auction_id"
	restBidder    = "bidder"
	restBid       = "bid"
	restLot       = "lot"
)

// RegisterRoutes - Central function to define routes that get registered by the main application
func RegisterRoutes(cliCtx context.CLIContext, r *mux.Router) {
	r.HandleFunc(
		fmt.Sprintf("/auction/getauctions"), queryGetAuctionsHandlerFn(cliCtx),
	).Methods("GET")

	r.HandleFunc(
		fmt.Sprintf("/auction/bid/{%s}/{%s}/{%s}/{%s}", restAuctionID, restBidder, restBid, restLot),
		bidHandlerFn(cliCtx),
	).Methods("PUT")
}

func queryGetAuctionsHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, _, err := cliCtx.QueryWithData("/custom/auction/getauctions", nil)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusNotFound, err.Error())
			return
		}
		rest.PostProcessResponse(w, cliCtx, res)
	}
}

func bidHandlerFn(cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req placeBidReq
		vars := mux.Vars(r)
		strAuctionID := vars[restAuctionID]
		bechBidder := vars[restBidder]
		strBid := vars[restBid]
		strLot := vars[restLot]

		auctionID, err := auction.NewIDFromString(strAuctionID)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		bidder, err := sdk.AccAddressFromBech32(bechBidder)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		bid, err := sdk.ParseCoin(strBid)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		lot, err := sdk.ParseCoin(strLot)
		if err != nil {
			rest.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := auction.NewMsgPlaceBid(auctionID, bidder, bid, lot)
		clientrest.WriteGenerateStdTxResponse(w, cliCtx, req.BaseReq, []sdk.Msg{msg})

	}
}
