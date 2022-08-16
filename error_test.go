package vk_sdk

func (e *apiError) fillRandomly() {
	subcode := randInt()
	l := randIntn(maxArrayLength + 1)
	requestParams := make([]RequestParam, l)
	for i := 0; i < l; i++ {
		requestParams[i].fillRandomly()
	}
	redirectURI := randString()
	confirmationText := randString()
	captchaSID := randString()
	captchaImg := randString()

	*e = apiError{
		ErrorCode:    randInt(),
		ErrorSubcode: &subcode,
		ErrorMsg:     randString(),
		ErrorText:    randString(),
		ReqParams:    requestParams,
		RedirURI:     &redirectURI,
		ConfirmText:  &confirmationText,
		CaptchaSID:   &captchaSID,
		CaptchaImg:   &captchaImg,
	}
}

func (p *RequestParam) fillRandomly() {
	p.Key = randString()
	p.Value = randString()
}
