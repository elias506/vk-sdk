package vk_sdk

type apiError struct {
	ErrorCode    int            `json:"error_code"`
	ErrorSubcode *int           `json:"error_subcode,omitempty"`
	ErrorMsg     string         `json:"error_msg"`
	ErrorText    string         `json:"error_text"`
	ReqParams    []RequestParam `json:"request_params"`
	RedirURI     *string        `json:"redirect_uri,omitempty"`
	ConfirmText  *string        `json:"confirmation_text,omitempty"`
	// Captcha fields
	CaptchaSID *string `json:"captcha_sid,omitempty"`
	CaptchaImg *string `json:"captcha_img,omitempty"`
}

type RequestParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Code returns error code.
func (e apiError) Code() int {
	return e.ErrorCode
}

// Subcode returns error subcode.
func (e apiError) Subcode() *int {
	return e.ErrorSubcode
}

// Msg returns error message.
func (e apiError) Msg() string {
	return e.ErrorMsg
}

// Text returns error text.
func (e apiError) Text() string {
	return e.ErrorText
}

func (e apiError) RequestParams() []RequestParam {
	return e.ReqParams
}

func (e apiError) RedirectURI() *string {
	return e.RedirURI
}

func (e apiError) ConfirmationText() *string {
	return e.ConfirmText
}

func (e apiError) Is(code ErrorCode) bool {
	return e.ErrorCode == int(code)
}

func (e apiError) Captcha() Captcha {
	if e.CaptchaImg == nil && e.CaptchaSID == nil {
		return nil
	}

	c := new(captcha)

	if e.CaptchaSID != nil {
		c.CaptchaSID = *e.CaptchaSID
	}

	if e.CaptchaImg != nil {
		c.CaptchaImg = *e.CaptchaImg
	}

	return c
}

type captcha struct {
	CaptchaSID string
	CaptchaImg string
}

func (c captcha) SID() string {
	return c.CaptchaSID
}

func (c captcha) Img() string {
	return c.CaptchaImg
}
