package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rawen554/goph-keeper/cmd/client/internal/client"
	"github.com/rawen554/goph-keeper/internal/models"
	"github.com/spf13/viper"
)

func Register(login string, password string) (creds *models.TokenResponse, err error) {
	httpclient := client.GetHttpClient()
	if httpclient == nil {
		return nil, fmt.Errorf("configuration error")
	}
	endpoint, _ := url.JoinPath(httpclient.ApiURL, "api/user/register")

	b, _ := json.Marshal(LoginReq{Login: login, Password: password})

	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("error: %w\n", err)
	}
	// в заголовках запроса указываем кодировку
	request.Header.Add("Content-Type", "application/json")
	// отправляем запрос и получаем ответ
	response, err := httpclient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error: %w\n", err)
	}
	// выводим код ответа
	fmt.Println("Статус-код ", response.Status)
	defer func() {
		if err := response.Body.Close(); err != nil {
			fmt.Printf("error: %w\n", err)
		}
	}()

	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Error in Register")
	}

	creds = &models.TokenResponse{}
	if err = json.NewDecoder(response.Body).Decode(creds); err != nil {
		return nil, fmt.Errorf("error decode body: %w\n", err)
	}

	viper.Set("api", httpclient.ApiURL)

	return creds, nil
}
