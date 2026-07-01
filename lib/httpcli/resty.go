package httpcli

import "github.com/go-resty/resty/v2"

var client = resty.New().SetDebug(true)

type HttpClient struct {
	ReqHeaders map[string]string
	URL        string
	ReqBody    interface{}
}

func Post(httpClient *HttpClient) (res *resty.Response, err error) {
	res, err = client.R().SetHeaders(httpClient.ReqHeaders).SetBody(httpClient.ReqBody).Post(httpClient.URL)
	return
}
