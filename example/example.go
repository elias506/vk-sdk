package main

import (
	"context"
	"fmt"
	"github.com/elias506/vk-sdk"
	"log"
	"net/http"
	"net/url"
)

func main() {
	// Build auth request
	authReq := vk_sdk.ImplicitFlowUserRequest{
		ClientID:    "<your_id>",
		RedirectURI: "",
		Display:     nil,
		Scope:       nil,
		State:       nil,
		Revoke:      false,
	}

	// get redirect url for user
	redirectURL := vk_sdk.GetAuthRedirectURL(authReq)

	fmt.Println("redirect url:", redirectURL.String())
	fmt.Print("paste new redirect here: ")

	var newRedirect string
	fmt.Scan(&newRedirect)

	// Parse url, that u could get from incoming http.Request.URL
	u, err := url.Parse(newRedirect)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Get UserToken from URL
	token, oAuthErr, err := vk_sdk.GetImplicitFlowUserToken(u)

	if err != nil {
		log.Fatal(err.Error())
	}
	if oAuthErr != nil {
		log.Fatal(oAuthErr.Error(), oAuthErr.Description())
	}

	// Set token and do request
	vk := vk_sdk.NewVK(http.DefaultClient)

	vk.SetToken(token.AccessToken())

	req := vk_sdk.Friends_GetOnline_Request{}

	resp, apiErr, err := vk.Friends_GetOnline(context.Background(), req, vk_sdk.TestMode())

	if err != nil {
		log.Fatal(err.Error())
	}

	if apiErr != nil {
		log.Fatal(apiErr.Code(), apiErr.Msg(), apiErr.RequestParams())
	}

	fmt.Println("Online friends IDs:", resp.Response)
}
