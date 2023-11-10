package client

import (
	"crypto/tls"
	"log"
	"net/http"
	"sync"

	"github.com/rawen554/goph-keeper/internal/logger"
	"github.com/spf13/viper"
)

type httpClientInstance struct {
	*http.Client
	APIURL string
}

var (
	httpClient *httpClientInstance
	once       sync.Once
)

func GetHTTPClient() *httpClientInstance {
	once.Do(
		func() {
			logger, err := logger.NewLogger()
			if err != nil {
				log.Fatal(err)
			}

			apiURL := viper.GetString("api")
			if apiURL == "" {
				logger.Errorln("empty API URL")
				httpClient = nil
				return
			}
			httpClient = &httpClientInstance{
				Client: &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					}},
				APIURL: apiURL,
			}
		})

	return httpClient
}
