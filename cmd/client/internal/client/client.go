package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"

	"github.com/spf13/viper"
)

type httpClientInstance struct {
	*http.Client
	ApiURL string
}

var (
	httpClient *httpClientInstance
	once       sync.Once
)

func GetHttpClient() *httpClientInstance {
	once.Do(
		func() {
			apiURL := viper.GetString("api")
			if apiURL == "" {
				fmt.Println("empty API URL")
				httpClient = nil
				return
			}
			httpClient = &httpClientInstance{
				Client: &http.Client{Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				}},
				ApiURL: apiURL,
			}
		})

	return httpClient
}
