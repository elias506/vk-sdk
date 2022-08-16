package vk_sdk

import (
	"golang.org/x/time/rate"
	"net/http"
)

// NewLimitClient is http.Client with requests per second limit
//
// VKontakte API methods (except methods from the secure and ads sections)
// with a user access key (UserToken.AccessToken) can be accessed no more than 3 times per second.
// For a community access key (GroupToken.AccessToken), the limit is 20 requests per second.
// If the logic of your application involves calling several methods in a row,
// it makes sense to pay attention to the execute method.
// It allows you to make up to 25 calls to different methods within a single request.
//
// For a service access key, the limits are as follows:
//    up to 10^4     - 5 rps
//    up to 10^5     - 20 rps
//    up to 5*10^5   - 40 rps
//    up to 10^6     - 50 rps
//    more than 10^6 - 60 rps
//
// The ads section methods have their own restrictions (https://dev.vk.com/method/ads).
//
// The maximum number of calls to secure section methods depends on the number of users who have installed the application.
// If the application is installed by less than 10,000 people, then you can make 5 requests per second,
// up to 100,000 - 8 requests, up to 1,000,000 - 20 requests, more than 1 million - 35 requests per second.
// If you exceed the frequency limit, the server will return an error with code 6: "Too many requests per second.".
//
// https://dev.vk.com/api/api-requests

type LimitClient http.Client

func NewLimitClient(limit, bursts int) *LimitClient {
	return (*LimitClient)(&http.Client{
		Transport: NewLimitTransport(limit, bursts),
	})
}

func (lc *LimitClient) SetLimit(limit, bursts int) {
	lc.Transport = &LimitTransport{
		defaultTransport: lc.Transport,
		limiter:          rate.NewLimiter(rate.Limit(limit), bursts),
	}
}

func NewLimitTransport(limit, bursts int) *LimitTransport {
	return &LimitTransport{
		defaultTransport: http.DefaultTransport,
		limiter:          rate.NewLimiter(rate.Limit(limit), bursts),
	}
}

type LimitTransport struct {
	defaultTransport http.RoundTripper
	limiter          *rate.Limiter
}

func (lt *LimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := lt.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}
	return lt.defaultTransport.RoundTrip(req)
}
