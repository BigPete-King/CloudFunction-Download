package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/tencentyun/scf-go-lib/cloudfunction"
	"github.com/tencentyun/scf-go-lib/events"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type DownloadParam struct {
	URL string `json：url`
}

func download(ctx context.Context, request events.APIGatewayRequest) (events.APIGatewayResponse, error) {

	domain := os.Getenv("COS_DOMAIN")
	secretID := os.Getenv("COS_SECRETID")
	secretKey := os.Getenv("COS_SECRETKEY")

	var downloadURL string
	if request.Body != "" {
		var body DownloadParam
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return events.APIGatewayResponse{}, err
		}
		downloadURL = body.URL
	}

	if downloadURL == "" {
		return events.APIGatewayResponse{}, errors.New("缺少URL参数")
	}

	resp, err := http.Get(downloadURL)
	if err != nil {
		return events.APIGatewayResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return events.APIGatewayResponse{}, errors.New("下载失败")
	}

	var fileName string
	if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != "" {
		if r, err := regexp.Compile("filename=(.*)"); err == nil {
			if subMatch := r.FindStringSubmatch(contentDisposition); subMatch != nil {
				fileName = subMatch[1]
			}
		}
	}

	if fileName == "" {
		if pos := strings.Index(downloadURL, "?"); pos != -1 {
			fileName = downloadURL[strings.LastIndex(downloadURL, "/")+1 : pos]
		} else {
			fileName = downloadURL[strings.LastIndex(downloadURL, "/")+1:]
		}
	}
	fmt.Println("fileName:" + fileName)
	bucketURL, err := url.Parse(domain)
	if err != nil {
		return events.APIGatewayResponse{}, err
	}
	baseURL := &cos.BaseURL{BucketURL: bucketURL}
	client := cos.NewClient(baseURL, &http.Client{
		//设置超时时间
		Timeout: 900 * time.Second,
		Transport: &cos.AuthorizationTransport{
			//如实填写账号和密钥，也可以设置为环境变量
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})
	_, err = client.Object.Put(context.Background(), fileName, resp.Body, nil)
	if err != nil {
		return events.APIGatewayResponse{}, err
	}
	result, _ := json.Marshal(DownloadParam{
		URL: domain + "/" + fileName,
	})

	return events.APIGatewayResponse{
		IsBase64Encoded: false,
		StatusCode:      200,
		Headers:         map[string]string{"Content-Type": "application/json"},
		Body:            string(result),
	}, nil
}

func main() {

	// Make the handler available for Remote Procedure Call by Cloud Function
	cloudfunction.Start(download)

}
