package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/rawen554/goph-keeper/internal/models"
	"go.uber.org/zap"
)

const OrdersAPI = "/api/orders/"

var ErrNoOrder = errors.New("order is not processed")
var ErrServiceBusy = errors.New("accrual is busy")

var NumberRegExp = regexp.MustCompile(`(\d+)`)

type ServiceBusyError struct {
	Err      error
	CoolDown time.Duration
	MaxRPM   int
}

func (sbe *ServiceBusyError) Error() string {
	return fmt.Sprintf("wait: %vs; max rpm: %v; %v", sbe.CoolDown.Seconds(), sbe.MaxRPM, sbe.Err)
}

func NewServiceBusyError(cooldown time.Duration, rpm int, err error) error {
	return &ServiceBusyError{
		CoolDown: cooldown,
		MaxRPM:   rpm,
		Err:      err,
	}
}

type AccrualClient struct {
	client      *retryablehttp.Client
	logger      *zap.SugaredLogger
	accrualAddr string
}

type Accrual interface {
	GetOrderInfo(num string) (*AccrualOrderInfoShema, error)
}

type AccrualOrderInfoShema struct {
	Order   string        `json:"order"`
	Status  models.Status `json:"status"`
	Accrual float64       `json:"accrual,omitempty"`
}

func NewAccrualClient(accrualAddr string, logger *zap.SugaredLogger) (Accrual, error) {
	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.CheckRetry = checkRetry
	client.Backoff = backoff

	return &AccrualClient{
		accrualAddr: accrualAddr,
		client:      client,
		logger:      logger,
	}, nil
}

func (a *AccrualClient) GetOrderInfo(num string) (*AccrualOrderInfoShema, error) {
	url, err := url.JoinPath(a.accrualAddr, OrdersAPI, num)
	if err != nil {
		return nil, fmt.Errorf("error joining path: %w", err)
	}

	req, err := retryablehttp.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	result, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting order info from accrual: %w", err)
	}

	res, err := io.ReadAll(result.Body)

	defer func() {
		if err := result.Body.Close(); err != nil {
			a.logger.Errorf("error close body: %w", err)
		}
	}()

	switch result.StatusCode {
	case http.StatusOK:
		var orderInfo AccrualOrderInfoShema

		if err := json.Unmarshal(res, &orderInfo); err != nil {
			return nil, fmt.Errorf("error parsing json: %w", err)
		}

		return &orderInfo, nil
	case http.StatusNoContent:
		return nil, ErrNoOrder
	case http.StatusTooManyRequests:
		cooldown, err := strconv.Atoi(result.Header.Get("Retry-After"))
		if err != nil {
			return nil, fmt.Errorf("error converting header Retry-After: %w", err)
		}

		rpm := NumberRegExp.Find(res)
		if rpm == nil {
			return nil, fmt.Errorf("not found MaxRPM in body: %w", err)
		}
		preparedRPM, err := strconv.Atoi(string(rpm))
		if err != nil {
			return nil, fmt.Errorf("cant convert MaxRPM to int: %w", err)
		}

		return nil, NewServiceBusyError(time.Duration(cooldown)*time.Second, preparedRPM, err)
	default:
		return nil, fmt.Errorf("unknown exception: %w", err)
	}
}

func checkRetry(ctx context.Context, res *http.Response, err error) (bool, error) {
	check, err := retryablehttp.DefaultRetryPolicy(ctx, res, err)
	if err != nil {
		return false, fmt.Errorf("accrual error in default retry policy : %w", err)
	}
	return check, nil
}

func backoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	return retryablehttp.LinearJitterBackoff(min, max, attemptNum, resp)
}
