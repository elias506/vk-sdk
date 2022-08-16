// go test -bench=Benchmark_Medium -benchmem
package main

import (
	"bytes"
	"context"
	"encoding/json"
	api2 "github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
	"github.com/elias506/vk-sdk"
	goVKAPI "github.com/go-vk-api/vk"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

const token = "cdb3af9beae43801707049269a35fe661efc88de8ae99d084edb7ded88b6694e37386cda7fe79e9943dd9"

type TestRoundTrip func(*http.Request) (*http.Response, error)

func (f TestRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func NewBenchClientMedium() *http.Client {
	name := "Вася"
	lastName := "Васильев"
	resp := vk_sdk.Users_Get_Response{
		Response: []vk_sdk.Users_UserFull{
			{
				Users_User: vk_sdk.Users_User{
					Users_UserMin: vk_sdk.Users_UserMin{
						Id:        1234567,
						FirstName: &name,
						LastName:  &lastName,
					},
				},
			},
		},
	}

	j, _ := json.Marshal(resp)

	h := http.Header{
		"Content-Type": []string{"application/json"},
	}

	t := TestRoundTrip(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     h,
			Body:       ioutil.NopCloser(bytes.NewBuffer(j)),
		}, nil
	})

	return &http.Client{Transport: t}
}

func Benchmark_Medium_VK_SDK(b *testing.B) {
	vk := vk_sdk.NewVK(NewBenchClientMedium(), token)

	for n := 0; n < b.N; n++ {
		ctx := context.Background()
		req := vk_sdk.Users_Get_Request{
			UserIds: &[]string{"elias506"},
			Fields: &[]vk_sdk.Users_Fields{
				vk_sdk.Users_Fields_Sex,
				vk_sdk.Users_Fields_City,
				vk_sdk.Users_Fields_Country,
				vk_sdk.Users_Fields_Online,
				vk_sdk.Users_Fields_HasMobile,
				vk_sdk.Users_Fields_Nickname,
				vk_sdk.Users_Fields_Music,
				vk_sdk.Users_Fields_About,
			},
			NameCase: nil,
		}

		resp, apiErr, err := vk.Users_Get(ctx, req, vk_sdk.TestMode())

		if err != nil {
			panic(err)
		}

		if apiErr != nil {
			panic("")
		}

		if *resp.Response[0].FirstName != "Вася" {
			panic("")
		}
	}
}

func Benchmark_Medium_SevereCloud(b *testing.B) {
	vk2 := api2.NewVK(token)
	vk2.Client = NewBenchClientMedium()
	vk2.Limit = 0

	for n := 0; n < b.N; n++ {
		builder := params.NewUsersGetBuilder()

		builder.UserIDs([]string{"elias506"})
		builder.Fields([]string{"sex", "city", "country", "online", "has_mobile", "nickname", "music", "about"})
		builder.TestMode(true)

		resp, err := vk2.UsersGet(builder.Params)

		if err != nil {
			panic("")
		}

		if resp[0].FirstName != "Вася" {
			panic("")
		}
	}
}

func Benchmark_Medium_goVKAPI(b *testing.B) {
	vk, err := goVKAPI.NewClientWithOptions(
		goVKAPI.WithToken(os.Getenv(token)),
		goVKAPI.WithHTTPClient(NewBenchClientMedium()))

	if err != nil {
		panic(err.Error())
	}

	for n := 0; n < b.N; n++ {
		var resp []vk_sdk.Users_UserFull

		err = vk.CallMethod("users.get", goVKAPI.RequestParams{
			"user_id":   []string{"elias506"},
			"fields":    []string{"sex", "city", "country", "online", "has_mobile", "nickname", "music", "about"},
			"test_mode": 1,
		}, &resp)

		if err != nil {
			panic(err.Error())
		}

		if resp[0].FirstName != nil && *resp[0].FirstName != "Вася" {
			panic("")
		}
	}
}
