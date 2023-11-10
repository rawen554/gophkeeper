package logic

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rawen554/goph-keeper/cmd/client/internal/client"
	"github.com/rawen554/goph-keeper/internal/models"
	"github.com/rawen554/goph-keeper/internal/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func SaveOrUpdateData(logger *zap.SugaredLogger, data *models.DataRecord) error {
	login := viper.GetString("login")
	if login == "" {
		err := fmt.Errorf("not logged in")
		logger.Error(err)
		return err
	}

	if err := utils.CreateUsersDir(login); err != nil {
		err = fmt.Errorf("error creating users dir: %w", err)
		logger.Error(err)
		return err
	}

	ext := utils.GetExtension(data.Type)
	filename := fmt.Sprintf("%s%s", data.Name, ext)
	filepath := filepath.Join(".", login, filename)
	localFile, err := os.OpenFile(filepath, os.O_RDONLY, 0600)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			wrLocalFile, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
			if err != nil {
				return err
			}

			defer wrLocalFile.Close()

			if err := json.NewEncoder(wrLocalFile).Encode(data); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	var localData models.DataRecord
	if err := json.NewDecoder(localFile).Decode(&localData); err != nil {
		return err
	}

	if err := localFile.Close(); err != nil {
		return err
	}

	wrLocalFile, err := os.OpenFile(filepath, os.O_RDWR, 0600)

	if data.ID != 0 && localData.ID == 0 {
		wrLocalFile.Truncate(0)
		if err := json.NewEncoder(wrLocalFile).Encode(data); err != nil {
			return err
		}
	}

	return nil
}

func GetRecord(ctx context.Context, name string) (*models.DataRecord, error) {
	token := viper.GetString("token")
	if token == "" {
		return nil, fmt.Errorf("No auth data, login first")
	}

	httpclient := client.GetHTTPClient()
	if httpclient == nil {
		return nil, fmt.Errorf("configuration error")
	}
	endpoint, _ := url.JoinPath(httpclient.APIURL, "api/user/records", name)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := httpclient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error: %w", err)
	}

	if response.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("record not found")
	}

	var record models.DataRecord
	if err = json.NewDecoder(response.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("error decode body: %w", err)
	}

	return &record, nil
}

func PutRecord(ctx context.Context, args []string) (*models.DataRecord, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("bad request")
	}

	dataType := strings.ToLower(args[0])
	var data string

	switch dataType {
	case "pass":
		data = args[1]
	default:
		path := args[1]
		fi, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		data = fi.Name()
	}

	token := viper.GetString("token")
	if token == "" {
		return nil, fmt.Errorf("No auth data, login first")
	}

	httpclient := client.GetHTTPClient()
	if httpclient == nil {
		return nil, fmt.Errorf("configuration error")
	}
	endpoint, _ := url.JoinPath(httpclient.APIURL, "api/user/records")

	checksum := fmt.Sprintf("%x", md5.Sum([]byte(data)))

	dataObj := models.DataRecordRequest{
		Type:     models.DataType(dataType),
		Name:     args[2],
		Data:     data,
		Checksum: checksum,
	}
	dataObjB, err := json.Marshal(dataObj)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(dataObjB))
	if err != nil {
		return nil, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := httpclient.Do(request)
	if err != nil {
		return &models.DataRecord{
			Data:     dataObj.Data,
			Checksum: dataObj.Checksum,
			Type:     dataObj.Type,
			Name:     dataObj.Name,
		}, err
	}

	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("error in Post data")
	}

	var record models.DataRecord
	if err = json.NewDecoder(response.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("error decode body: %w", err)
	}

	return &record, nil
}

func ListRecords(ctx context.Context, logger *zap.SugaredLogger) ([]models.DataRecord, error) {
	token := viper.GetString("token")
	if token == "" {
		err := fmt.Errorf("no auth data, login first")
		logger.Error(err)
		return nil, err
	}

	httpclient := client.GetHTTPClient()
	if httpclient == nil {
		err := fmt.Errorf("configuration error")
		logger.Error(err)
		return nil, err
	}
	endpoint, _ := url.JoinPath(httpclient.APIURL, "api/user/records")

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		logger.Error(err)
		return nil, err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := httpclient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode == http.StatusNoContent {
		logger.Infoln("no records found")
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error in listrecords\n")
	}

	records := make([]models.DataRecord, 0)
	if err = json.NewDecoder(response.Body).Decode(&records); err != nil {
		return nil, fmt.Errorf("error decode body: %w\n", err)
	}

	return records, nil
}

func SyncDataRecords(ctx context.Context, logger *zap.SugaredLogger) error {
	records, err := ListRecords(ctx, logger)
	if err != nil {
		return err
	}

	g := new(errgroup.Group)
	for _, r := range records {
		data := r

		g.Go(func() error {
			if err := SaveOrUpdateData(logger, &data); err != nil {
				return err
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
