// To run most API methods you need to pass an access_token, a special access key.
// Token is a string of digits and latin characters that may refers to a user, community or application itself.
//
// To get a token VK uses the OAuth 2.0 open protocol.
// Users do not send their login and password so accounts can not be compromised.

package vk_sdk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	oAuthScheme          = "https"
	oAuthHost            = "oauth.vk.com"
	oAuthPathAuthorize   = "authorize"
	oAuthPathAccessToken = "access_token"
)

// UserToken is required to run almost all API methods excepting the secure section.
// Some methods, such as VK.Users_Get, can be called without a token
// but some data may be unavailable because it does matter who exactly tries to get it.
//
// Token is a kind of user signature in the application.
// It reports the server which user sends requests and what permissions did they grant to the app.
//
// To get a user token use one of these ways:
// - Implicit flow to run methods on behalf of a user in Javascript apps and Standalone clients (mobile or desktop).
// - Authorization code flow to run methods on behalf of a user from the server side on a website.
type UserToken struct {
	accessToken string
	expiresIn   time.Duration
	userID      int
	email       *string
	state       *string
}

// AccessToken returns user access token.
func (ut UserToken) AccessToken() string {
	return ut.accessToken
}

// ExpiresIn returns token expiredIn time.
func (ut UserToken) ExpiresIn() time.Duration {
	return ut.expiresIn
}

// UserID returns user ID.
func (ut UserToken) UserID() int {
	return ut.userID
}

// Email will return the user's email, if the user has an email specified
// and application has requested the appropriate AccessPermission.
func (ut UserToken) Email() *string {
	return ut.email
}

// State returns request state.
func (ut UserToken) State() *string {
	return ut.state
}

// GroupToken allows working with API on behalf of a group, event or public page. It can be used to answer the community messages.
//
// Methods that have a special mark in the list can be called with a community token.
//
// There are three methods to get it:
// - Implicit flow. To run methods on behalf of a community in Javascript apps and standalone clients (desktop or mobile).
// - Authorization code flow. To work with API on behalf of a community from a website server.
// - At the community management page. Just open the "Manage community" tab, go to "API usage" tab and click "Create token".
//
// Getting a list of administered communities
//
// Only the administrator of a group can obtain group access key via OAuth.
// To obtain access keys for all or several user communities at once, we recommend adding this additional step to the authorization process.
// Get the user's access key (for working from the client or for working from the server) with the scope=groups rights and make a request to the VK.Groups_Get method with the Groups_Get_Request.Filter = Groups_Filter_Admin parameter to get a list of administered community IDs.
// Then use all or part of the received values as the group_ids parameter.
type GroupToken struct {
	accessToken string
	groupID     int
}

// AccessToken returns group access token.
func (gt GroupToken) AccessToken() string {
	return gt.accessToken
}

// GroupID returns group ID for the token.
func (gt GroupToken) GroupID() int {
	return gt.groupID
}

// GroupTokens contains access tokens with group IDs array and tokens expiredIn time.
type GroupTokens struct {
	tokens    []GroupToken
	expiresIn time.Duration
	state     *string
}

// Tokens returns tokens with group IDs.
func (gts GroupTokens) Tokens() []GroupToken {
	return gts.tokens
}

// ExpiresIn returns token expiredIn time.
func (gts GroupTokens) ExpiresIn() time.Duration {
	return gts.expiresIn
}

// State returns request state.
func (gts GroupTokens) State() *string {
	return gts.state
}

// OAuthError contains fields of OAuth error.
type OAuthError interface {
	// Error returns oAuth error.
	Error() string

	// Description returns oAuth error description.
	Description() string
}

type oAuthError struct {
	Err  string `json:"error"`
	Desc string `json:"error_description"`
}

type OAuthErrorType string

const (
	OAuthErrorTypeInvalidRequest          OAuthErrorType = "invalid_request"
	OAuthErrorTypeUnauthorizedClient      OAuthErrorType = "unauthorized_client"
	OAuthErrorTypeUnsupportedResponseType OAuthErrorType = "unsupported_response_type"
	OAuthErrorTypeInvalidScope            OAuthErrorType = "invalid_scope"
	OAuthErrorTypeServerError             OAuthErrorType = "server_error"
	OAuthErrorTypeTemporarilyUnavailable  OAuthErrorType = "temporarily_unavailable"
	OAuthErrorTypeAccessDenied            OAuthErrorType = "access_denied"
	OAuthErrorTypeInvalidGrant            OAuthErrorType = "invalid_grant"
	OAuthErrorTypeNeedValidation          OAuthErrorType = "need_validation"
	OAuthErrorTypeNeedCaptcha             OAuthErrorType = "need_captcha"
)

func (oAuthErr oAuthError) Error() string {
	return oAuthErr.Err
}

func (oAuthErr oAuthError) Description() string {
	return oAuthErr.Description()
}

func (oAuthErr oAuthError) Is(oAuthType OAuthErrorType) bool {
	return oAuthErr.Err == string(oAuthType)
}

// getOAuthErrorFromValues returns OAuthError if present in values.
func getOAuthErrorFromValues(values url.Values) OAuthError {
	if values.Has("error") {
		return &oAuthError{
			Err:  values.Get("error"),
			Desc: values.Get("error_description"),
		}
	}

	return nil
}

// authRequest is interface that implement all auth requests.
// Needed to easily create a redirect URL using GetAuthRedirectURL.
type authRequest interface {
	values() url.Values
}

// RedirectURL contains redirect url.
type RedirectURL struct {
	url *url.URL
}

// String returns valid redirect url.URL string.
func (u RedirectURL) String() string {
	return u.url.String()
}

const DefaultRedirectURI = "https://oauth.vk.com/blank.html"

// GetAuthRedirectURL build the URL to which the user's browser will be redirected after granting permissions.
// If the user is not authorized on VKontakte in the browser used,
// then in the dialog box he will be prompted to enter his username and password.
func GetAuthRedirectURL(req authRequest) RedirectURL {
	return RedirectURL{
		url: &url.URL{
			Scheme:   oAuthScheme,
			Host:     oAuthHost,
			Path:     oAuthPathAuthorize,
			RawQuery: req.values().Encode(),
		},
	}
}

type DisplayType string

const (
	// DisplayTypePage authorization form in a separate window.
	DisplayTypePage DisplayType = "page"
	// DisplayTypePopup a pop-up window.
	DisplayTypePopup DisplayType = "popup"
	// DisplayTypeMobile authorization for mobile devices (uses no Javascript).
	DisplayTypeMobile DisplayType = "mobile"
)

const (
	// responseTypeToken is used for getting tokens by Implicit Flow.
	responseTypeToken = "token"
	// responseTypeCode is used for getting tokens by Authorization Code Flow.
	responseTypeCode = "code"
)

// ImplicitFlowUserRequest is request to build Implicit Flow redirect URL by GetAuthRedirectURL to get UserToken.
//
// https://dev.vk.com/api/access-token/implicit-flow-user
type ImplicitFlowUserRequest struct {
	// Application id.
	ClientID string
	// Address to redirect user after authorization.
	RedirectURI string
	// Sets authorization page appearance.
	Display *DisplayType
	// Permissions bit mask, to check on authorization and request if necessary.
	Scope *[]AccessPermission
	// Response type to receive.
	// NOTE:
	// If you pass a pointer to an empty string, Vkontakte API will not return the state field
	// for token and UserToken.State will be empty pointer. Actual for Version==5.131
	State *string
	// Sets that permissions request should not be skipped even if a user is already authorized.
	Revoke bool
}

// values build url.Values to get redirect URL by GetAuthRedirectURL
func (req ImplicitFlowUserRequest) values() url.Values {
	values := make(url.Values, 7)

	setString(values, "response_type", responseTypeToken)

	setString(values, "client_id", req.ClientID)
	setString(values, "redirect_uri", req.RedirectURI)
	if req.Display != nil {
		setString(values, "display", string(*req.Display))
	}
	if req.Scope != nil {
		scope := 0
		for _, permission := range *req.Scope {
			scope += int(permission)
		}
		setInt(values, "scope", scope)
	}
	if req.State != nil {
		setString(values, "state", *req.State)
	}
	if req.Revoke {
		setInt(values, "revoke", 1)
	}

	return values
}

// GetImplicitFlowUserToken returns a UserToken for calling VK API methods directly from the user's device.
// An access key obtained in this way cannot be used for requests from the server.
//
// https://dev.vk.com/api/access-token/implicit-flow-user
func GetImplicitFlowUserToken(u *url.URL) (UserToken, OAuthError, error) {
	var token UserToken

	values, err := url.ParseQuery(u.Fragment)

	if err != nil {
		return token, nil, err
	}

	if oAuthErr := getOAuthErrorFromValues(values); oAuthErr != nil {
		return token, oAuthErr, nil
	}

	expiresIn, err := strconv.Atoi(values.Get("expires_in"))

	if err != nil {
		return token, nil, err
	}

	userID, err := strconv.Atoi(values.Get("user_id"))

	if err != nil {
		return token, nil, err
	}

	token.accessToken = values.Get("access_token")
	token.expiresIn = time.Duration(expiresIn) * time.Second
	token.userID = userID
	if values.Has("email") {
		existEmail := values.Get("email")
		token.email = &existEmail
	}
	if values.Has("state") {
		existState := values.Get("state")
		token.state = &existState
	}

	return token, nil, nil
}

// ImplicitFlowGroupRequest is request to build Implicit Flow redirect URL by GetAuthRedirectURL to get GroupTokens.
//
// https://vk.com/dev/implicit_flow_group
type ImplicitFlowGroupRequest struct {
	// Application id.
	ClientID string
	// Address to redirect user after authorization.
	RedirectURI string
	// Group IDs for which you need to get an access key.
	// The parameter must be a string containing values without a minus sign.
	GroupIDs []int
	// Sets authorization page appearance.
	Display *DisplayType
	// Permissions bit mask, to check on authorization and request if necessary.
	Scope *[]AccessPermission
	// An arbitrary string that will be returned together with authorization result.
	// NOTE:
	// If you pass a pointer to an empty string, Vkontakte API will not return the state field
	// for token and UserToken.State will be empty pointer. Actual for Version==5.131
	State *string
}

// values build url.Values to get redirect URL by GetAuthRedirectURL.
func (req ImplicitFlowGroupRequest) values() url.Values {
	values := make(url.Values, 8)

	setString(values, "response_type", responseTypeToken)
	setString(values, "v", Version)

	setString(values, "client_id", req.ClientID)
	setString(values, "redirect_uri", req.RedirectURI)
	setInts(values, "group_ids", req.GroupIDs)
	if req.Display != nil {
		setString(values, "display", string(*req.Display))
	}
	if req.Scope != nil && len(*req.Scope) > 0 {
		scope := 0
		for _, permission := range *req.Scope {
			scope += int(permission)
		}
		setInt(values, "scope", scope)
	}
	if req.State != nil {
		setString(values, "state", *req.State)
	}

	return values
}

// GetImplicitFlowGroupTokens returns GroupTokens to call VK API methods directly from the user's device.
//
// https://dev.vk.com/api/access-token/implicit-flow-community
func GetImplicitFlowGroupTokens(u *url.URL) (GroupTokens, OAuthError, error) {
	var tokens GroupTokens

	values, err := url.ParseQuery(u.Fragment)

	if err != nil {
		return tokens, nil, err
	}

	if oAuthErr := getOAuthErrorFromValues(values); oAuthErr != nil {
		return tokens, oAuthErr, nil
	}

	expiresIn, err := strconv.Atoi(values.Get("expires_in"))

	if err != nil {
		return tokens, nil, err
	}

	tokens.expiresIn = time.Duration(expiresIn) * time.Second

	if values.Has("state") {
		existState := values.Get("state")
		tokens.state = &existState
	}

	// parse access tokens
	const tokenPrefix = "access_token_"
	for key, v := range values {
		if strings.HasPrefix(key, tokenPrefix) {

			groupID, err := strconv.Atoi(strings.TrimSuffix(key, tokenPrefix))

			if err != nil {
				return tokens, nil, err
			}

			tokens.tokens = append(tokens.tokens, GroupToken{
				accessToken: v[0],
				groupID:     groupID,
			})
		}
	}

	return tokens, nil, nil
}

// buildGetTokenRequest build request to get Authorization Code Flow token.
func buildGetTokenRequest(values url.Values) (*http.Request, error) {
	reqURL := &url.URL{
		Scheme:   oAuthScheme,
		Host:     oAuthHost,
		Path:     oAuthPathAccessToken,
		RawQuery: values.Encode(),
	}

	req, err := http.NewRequest("GET", reqURL.String(), nil)

	if err != nil {
		return nil, err
	}

	return req, nil
}

// AuthorizationCodeFlowUserRequest is request to build Authorization Code Flow redirect URL by GetAuthRedirectURL.
// and to do request by GetAuthCodeFlowUserToken to get UserToken.
//
// https://dev.vk.com/api/access-token/authcode-flow-user
type AuthorizationCodeFlowUserRequest struct {
	// Application id.
	ClientID string
	// Address to which the code will be sent (the domain of the specified address must match
	// the main domain in the application settings and listed values in
	// the list of trusted redirect uri, addresses are compared up to the path part).
	RedirectURI string
	// Sets authorization page appearance.
	Display *DisplayType
	// Permissions bit mask, to check on authorization and request if necessary.
	Scope *[]AccessPermission
	// An arbitrary string that will be returned together with authorization result.
	// NOTE:
	// If you pass a pointer to an empty string, Vkontakte API will not return the state field
	// for token and UserToken.State will be empty pointer. Actual for Version==5.131
	State *string
}

// values build url.Values to get redirect URL by GetAuthRedirectURL.
func (req AuthorizationCodeFlowUserRequest) values() url.Values {
	values := make(url.Values, 6)

	setString(values, "response_type", responseTypeCode)

	setString(values, "client_id", req.ClientID)
	setString(values, "redirect_uri", req.RedirectURI)
	if req.Display != nil {
		setString(values, "display", string(*req.Display))
	}
	if req.Scope != nil && len(*req.Scope) > 0 {
		scope := 0
		for _, permission := range *req.Scope {
			scope += int(permission)
		}
		setInt(values, "scope", scope)
	}
	if req.State != nil {
		setString(values, "state", *req.State)
	}

	return values
}

// userTokenUnmarshaler is struct with public fields for unmarshalling json fields to UserToken from API response.
type userTokenUnmarshaler struct {
	AccessToken string  `json:"access_token"`
	ExpiresIn   int     `json:"expires_In"`
	UserID      int     `json:"user_id"`
	Email       *string `json:"email"`
}

// getUserTokenFromJSON returns UserToken if present in JSON data by userTokenUnmarshaler.
func getUserTokenFromJSON(data []byte, state *string) (UserToken, error) {
	var (
		token       UserToken
		unmarshaler userTokenUnmarshaler
	)

	if err := json.Unmarshal(data, &unmarshaler); err != nil {
		return token, err
	}

	token.accessToken = unmarshaler.AccessToken
	token.expiresIn = time.Duration(unmarshaler.ExpiresIn) * time.Second
	token.userID = unmarshaler.UserID
	token.email = unmarshaler.Email
	token.state = state

	return token, nil
}

// getUserTokenFromResponse parse http.Response from Authorization Code Flow request
// and returns UserToken with possible OAuthError and error.
//
// https://dev.vk.com/api/access-token/authcode-flow-user
func getUserTokenFromResponse(resp *http.Response, state *string) (token UserToken, OAuthErr OAuthError, err error) {
	defer func() {
		if closeErr := resp.Body.Close(); err == nil {
			err = closeErr
		}
	}()

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return token, nil, err
	}

	var oAuthErr oAuthError
	if err = json.Unmarshal(respBody, &oAuthErr); oAuthErr.Err != "" || err != nil {
		return token, oAuthErr, err
	}

	token, err = getUserTokenFromJSON(respBody, state)

	return
}

// GetAuthCodeFlowUserToken returns to UserToken call VK API methods from the server side of your application.
// An access key obtained in this way is not tied to an IP address,
// but the set of rights that an application can obtain is limited for security reasons.
// NOTE:
// Incoming URL expires 1 hour after the user is authorized on it.
//
// https://dev.vk.com/api/access-token/authcode-flow-user
func GetAuthCodeFlowUserToken(u *url.URL, client *http.Client, req AuthorizationCodeFlowUserRequest, clientSecret string) (UserToken, OAuthError, error) {
	values, err := url.ParseQuery(u.Fragment)

	if err != nil {
		return UserToken{}, nil, err
	}

	if oAuthErr := getOAuthErrorFromValues(values); oAuthErr != nil {
		return UserToken{}, oAuthErr, nil
	}

	// build new request values to get token
	newValues := make(url.Values, 4)

	newValues.Set("client_id", req.ClientID)
	newValues.Set("client_secret", clientSecret)
	newValues.Set("redirect_uri", req.RedirectURI)
	newValues.Set("code", values.Get("code"))

	httpReq, err := buildGetTokenRequest(newValues)

	if err != nil {
		return UserToken{}, nil, err
	}

	resp, err := client.Do(httpReq)

	if err != nil {
		return UserToken{}, nil, err
	}

	// getting the state that comes with the code
	var state *string
	if values.Has("state") {
		existState := values.Get("state")
		state = &existState
	}

	return getUserTokenFromResponse(resp, state)
}

// AuthorizationCodeFlowGroupRequest is request to build Authorization Code Flow redirect URL by GetAuthRedirectURL
// and to do request by GetAuthCodeFlowGroupTokens to get GroupTokens.
//
// https://dev.vk.com/api/access-token/authcode-flow-community
type AuthorizationCodeFlowGroupRequest struct {
	// Application id.
	ClientID string
	// Address to which the code will be sent (the domain of the specified address must match
	// the main domain in the application settings and listed values in
	// the list of trusted redirect uri, addresses are compared up to the path part).
	RedirectURI string
	// Group IDs for which you need to get an access key.
	// The parameter must be a string containing values without a minus sign.
	GroupIDs []int
	// Sets authorization page appearance.
	Display *DisplayType
	// Permissions bit mask, to check on authorization and request if necessary.
	Scope *[]AccessPermission
	// An arbitrary string that will be returned together with authorization result.
	// NOTE:
	// If you pass a pointer to an empty string, Vkontakte API will not return the state field
	// for token and UserToken.State will be empty pointer. Actual for Version==5.131
	State *string
}

// values build url.Values to get redirect URL by GetAuthRedirectURL.
func (req AuthorizationCodeFlowGroupRequest) values() url.Values {
	values := make(url.Values, 8)

	setString(values, "response_type", responseTypeCode)
	setString(values, "v", Version)

	setString(values, "client_id", req.ClientID)
	setString(values, "redirect_uri", req.RedirectURI)
	setInts(values, "group_ids", req.GroupIDs)
	if req.Display != nil {
		setString(values, "display", string(*req.Display))
	}
	if req.Scope != nil && len(*req.Scope) > 0 {
		scope := 0
		for _, permission := range *req.Scope {
			scope += int(permission)
		}
		setInt(values, "scope", scope)
	}
	if req.State != nil {
		setString(values, "state", *req.State)
	}

	return values
}

// groupTokensUnmarshaler is struct with public fields for unmarshalling json data to GroupTokens from API response.
type groupTokensUnmarshaler struct {
	Groups    []groupTokenUnmarshaler `json:"groups"`
	ExpiresIn int                     `json:"expires_in"`
}

// groupTokenUnmarshaler is struct with public fields for unmarshalling json to GroupToken.
type groupTokenUnmarshaler struct {
	GroupID     int    `json:"group_id"`
	AccessToken string `json:"access_token"`
}

// getGroupTokensFromJSON returns GroupTokens if present in JSON data by groupTokensUnmarshaler and groupTokenUnmarshaler.
func getGroupTokensFromJSON(data []byte, state *string) (GroupTokens, error) {
	var (
		tokens      GroupTokens
		unmarshaler groupTokensUnmarshaler
	)

	if err := json.Unmarshal(data, &unmarshaler); err != nil {
		return tokens, err
	}

	tokens.expiresIn = time.Duration(unmarshaler.ExpiresIn) * time.Second
	tokens.state = state

	tokens.tokens = make([]GroupToken, 0, len(unmarshaler.Groups))

	for _, group := range unmarshaler.Groups {
		tokens.tokens = append(tokens.tokens, GroupToken{
			accessToken: group.AccessToken,
			groupID:     group.GroupID,
		})
	}

	return tokens, nil
}

// getGroupTokensFromResponse parse http.Response from Authorization Code Flow request
// and returns GroupTokens with possible OAuthError and error.
//
// https://dev.vk.com/api/access-token/authcode-flow-community
func getGroupTokensFromResponse(resp *http.Response, state *string) (token GroupTokens, OAuthErr OAuthError, err error) {
	defer func() {
		if closeErr := resp.Body.Close(); err == nil {
			err = closeErr
		}
	}()

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return token, nil, err
	}

	var oAuthErr oAuthError
	if err = json.Unmarshal(respBody, &oAuthErr); oAuthErr.Err != "" || err != nil {
		return token, oAuthErr, err
	}

	token, err = getGroupTokensFromJSON(respBody, state)

	return
}

// GetAuthCodeFlowGroupTokens returns GroupTokens to call VK API methods from the server side of your application.
// The access key obtained in this way is not tied to an IP address.
// NOTE:
// Incoming URL expires 1 hour after the user is authorized on it
//
// https://dev.vk.com/api/access-token/authcode-flow-community
func GetAuthCodeFlowGroupTokens(u *url.URL, client *http.Client, req AuthorizationCodeFlowUserRequest, clientSecret string) (GroupTokens, OAuthError, error) {
	values, err := url.ParseQuery(u.Fragment)

	if err != nil {
		return GroupTokens{}, nil, err
	}

	if oAuthErr := getOAuthErrorFromValues(values); oAuthErr != nil {
		return GroupTokens{}, oAuthErr, nil
	}

	// build new request values to get token
	newValues := make(url.Values, 4)

	newValues.Set("client_id", req.ClientID)
	newValues.Set("client_secret", clientSecret)
	newValues.Set("redirect_uri", req.RedirectURI)
	newValues.Set("code", values.Get("code"))

	httpReq, err := buildGetTokenRequest(newValues)

	if err != nil {
		return GroupTokens{}, nil, err
	}

	resp, err := client.Do(httpReq)

	if err != nil {
		return GroupTokens{}, nil, err
	}

	// getting the state that comes with the code
	var state *string
	if values.Has("state") {
		existState := values.Get("state")
		state = &existState
	}

	return getGroupTokensFromResponse(resp, state)
}
