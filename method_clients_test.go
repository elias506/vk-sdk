package vk_sdk

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

type TestRoundTrip func(*http.Request) (*http.Response, error)

func (f TestRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func NewTestClient(t *testing.T, token, methodName string, requestValues url.Values, responseBodyRaw []byte) *http.Client {
	transport := TestRoundTrip(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, apiScheme, req.URL.Scheme)
		assert.Equal(t, apiHost, req.URL.Host)
		assert.Equal(t, "/"+apiPath+"/"+methodName, req.URL.Path)

		reqBody, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		values, err := url.ParseQuery(string(reqBody))
		require.NoError(t, err)

		requestValues.Set(versionKey, Version)
		requestValues.Set(tokenKey, token)

		assert.EqualValues(t, requestValues, values)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBodyRaw)),
		}, nil
	})

	return &http.Client{
		Transport: transport,
	}
}

func NewApiErrorTestClient(t *testing.T, methodName string, errorBodyRaw []byte) *http.Client {
	transport := TestRoundTrip(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, apiScheme, req.URL.Scheme)
		assert.Equal(t, apiHost, req.URL.Host)
		assert.Equal(t, "/"+apiPath+"/"+methodName, req.URL.Path)

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioutil.NopCloser(bytes.NewBuffer(errorBodyRaw)),
		}, nil
	})

	return &http.Client{
		Transport: transport,
	}
}

func NewErrorTestClient(err error) *http.Client {
	transport := TestRoundTrip(func(req *http.Request) (*http.Response, error) {
		return nil, err
	})

	return &http.Client{
		Transport: transport,
	}
}
