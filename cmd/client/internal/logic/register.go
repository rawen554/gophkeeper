package logic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rawen554/goph-keeper/cmd/client/internal/client"
	"github.com/rawen554/goph-keeper/internal/models"
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
		panic(err)
	}
	// в заголовках запроса указываем кодировку
	request.Header.Add("Content-Type", "application/json")
	// отправляем запрос и получаем ответ
	response, err := httpclient.Do(request)
	if err != nil {
		panic(err)
	}
	// выводим код ответа
	fmt.Println("Статус-код ", response.Status)
	defer func() {
		if err := response.Body.Close(); err != nil {
			panic(err)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Error in Register")
	}

	creds = &models.TokenResponse{}
	if err = json.NewDecoder(response.Body).Decode(creds); err != nil {
		return nil, fmt.Errorf("error decode body: %w", err)
	}

	return creds, nil
}