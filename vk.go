package vk_sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type ApiError interface {
	// Code returns error code.
	Code() int

	// Subcode returns error subcode.
	Subcode() *int

	// Msg returns error message.
	Msg() string

	// Text returns error text
	Text() string

	// RequestParams returns error request parameters.
	RequestParams() []RequestParam

	// RedirectURI returns redirect URI
	RedirectURI() *string

	// ConfirmationText returns confirmation text
	ConfirmationText() *string

	// Captcha returns Captcha, if exist
	Captcha() Captcha

	// Is checks if the error code matches input ErrorCode
	Is(code ErrorCode) bool
}

type Captcha interface {
	SID() string
	Img() string
}

// VK the main structure for calling requests to the API
type VK struct {
	client *http.Client
	token  string
}

// NewVK create and return new VK
func NewVK(client *http.Client, token ...string) *VK {
	var vk VK

	vk.client = client

	if len(token) > 0 {
		vk.token = token[0]
	}

	return &vk
}

// SetToken set access token
func (vk *VK) SetToken(token string) {
	vk.token = token
}

func (vk *VK) doReq(methodName string, ctx context.Context, values url.Values, dst interface{}) (ApiError, error) {
	req, err := vk.buildRequest(methodName, ctx, values)

	if err != nil {
		return nil, err
	}

	resp, err := vk.client.Do(req)

	if err != nil {
		return nil, err
	}

	return vk.parseResponse(resp, dst)
}

const (
	apiScheme = "https"
	apiHost   = "api.vk.com"
	apiPath   = "method"

	versionKey = "v"
	tokenKey   = "access_token"
)

// buildRequest build request to Vkontakte API with version and access token
func (vk *VK) buildRequest(methodName string, ctx context.Context, values url.Values) (*http.Request, error) {
	values.Set(versionKey, Version)
	values.Set(tokenKey, vk.token)

	reqBody := bytes.NewBufferString(values.Encode())

	reqURL := &url.URL{
		Scheme: apiScheme,
		Host:   apiHost,
		Path:   apiPath + "/" + methodName,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL.String(), reqBody)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req, nil
}

// parseResponse parse response and returns Error if present
func (vk *VK) parseResponse(resp *http.Response, dst interface{}) (ApiErr ApiError, err error) {
	defer func() {
		if closeErr := resp.Body.Close(); err == nil {
			err = closeErr
		}
	}()

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	var apiErr apiError

	if err = json.Unmarshal(respBody, &apiErr); apiErr.ErrorCode != 0 || err != nil {
		return &apiErr, err
	}

	if err = json.Unmarshal(respBody, &dst); err != nil {
		return nil, err
	}

	return nil, nil
}
