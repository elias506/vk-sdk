package vk_sdk

import "strconv"

// Option is structure for optional method fields.
type Option struct {
	name  string
	value string
}

type Language int

const (
	Russian     Language = 0
	Ukrainian   Language = 1
	Belorussian Language = 2
	English     Language = 3
	Spanish     Language = 4
	Finnish     Language = 5
	Deutsch     Language = 6
	Italian     Language = 7
)

// Lang determines the Language for the data to be displayed on.
// For example country and city names.
// Numeric format from VK.Account_GetInfo is supported as well.
func Lang(l Language) Option {
	return Option{
		name:  "lang",
		value: strconv.Itoa(int(l)),
	}
}

// TestMode allows you to execute queries from a native application
// without enabling it for all users.
func TestMode() Option {
	return Option{
		name:  "test_mode",
		value: "1",
	}
}

// CaptchaSID received ID.
func CaptchaSID(sid string) Option {
	return Option{
		name:  "captcha_sid",
		value: sid,
	}
}

// CaptchaKey text input.
func CaptchaKey(key string) Option {
	return Option{
		name:  "captcha_key",
		value: key,
	}
}
