// go test -bench=Benchmark_Small -benchmem
package main

import (
	"bytes"
	"context"
	"encoding/json"
	severeCloud "github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
	"github.com/elias506/vk-sdk"
	goVKAPI "github.com/go-vk-api/vk"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func NewBenchClientSmall() *http.Client {
	resp := vk_sdk.Base_Ok_Response{
		Response: 1,
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

func Benchmark_Small_VK_SDK(b *testing.B) {
	vk := vk_sdk.NewVK(NewBenchClientSmall(), token)

	for n := 0; n < b.N; n++ {
		ctx := context.Background()
		comment := "Comment"
		req := vk_sdk.Users_Report_Request{
			UserId:  100,
			Type:    "Type",
			Comment: &comment,
		}

		resp, apiErr, err := vk.Users_Report(ctx, req)

		if err != nil {
			panic(err.Error())
		}

		if apiErr != nil {
			panic("")
		}

		if resp.Response != 1 {
			panic("")
		}
	}
}

func Benchmark_Small_SevereCloud(b *testing.B) {
	vk2 := severeCloud.NewVK(token)
	vk2.Client = NewBenchClientSmall()
	vk2.Limit = 0

	for n := 0; n < b.N; n++ {
		builder := params.NewUsersReportBuilder()

		builder.UserID(100)
		builder.Comment("Comment")
		builder.Type("Type")

		resp, err := vk2.UsersReport(builder.Params)

		if err != nil {
			panic(err.Error())
		}

		if resp != 1 {
			panic("")
		}
	}
}

func Benchmark_Small_goVKAPI(b *testing.B) {
	vk, err := goVKAPI.NewClientWithOptions(
		goVKAPI.WithToken(os.Getenv(token)),
		goVKAPI.WithHTTPClient(NewBenchClientSmall()))

	if err != nil {
		panic(err.Error())
	}

	for n := 0; n < b.N; n++ {
		var resp int

		err = vk.CallMethod("users.report", goVKAPI.RequestParams{
			"user_id": 100,
			"type":    "Type",
			"comment": "comment",
		}, &resp)

		if err != nil {
			panic(err.Error())
		}

		if resp != 1 {
			panic("")
		}
	}
}
