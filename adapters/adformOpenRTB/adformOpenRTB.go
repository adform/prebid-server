package adformOpenRTB

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the AdformOpenRTB adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	requestCopy := *request
	var validImps = make([]openrtb2.Imp, 0, len(requestCopy.Imp))

	for _, imp := range requestCopy.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var adformOpenRTBImpExt openrtb_ext.ExtImpAdformOpenRTB
		if err := json.Unmarshal(bidderExt.Bidder, &adformOpenRTBImpExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		imp.TagID = adformOpenRTBImpExt.MasterTagID.String()
		validImps = append(validImps, imp)
	}

	requestCopy.Imp = validImps

	requestJSON, err := json.Marshal(requestCopy)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
	}

	return []*adapters.RequestData{requestData}, errors
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: openrtb_ext.BidTypeNative,
			})
		}
	}

	return bidResponse, nil
}
