package processing

import (
	"context"
	"errors"
	"time"

	"github.com/rawen554/goph-keeper/internal/adapters/accrual"
	"github.com/rawen554/goph-keeper/internal/adapters/store"
	"github.com/rawen554/goph-keeper/internal/models"
	"go.uber.org/zap"
)

type ProcessingController struct {
	ordersChan   chan *models.Order
	store        store.Store
	accrual      accrual.Accrual
	logger       *zap.SugaredLogger
	cooldownChan chan time.Duration
}

const chanLen = 10

func NewProcessingController(
	store store.Store,
	accrual accrual.Accrual,
	logger *zap.SugaredLogger,
) *ProcessingController {
	ordersChan := make(chan *models.Order, chanLen)
	cooldownChan := make(chan time.Duration, 1)

	instance := &ProcessingController{
		ordersChan:   ordersChan,
		store:        store,
		accrual:      accrual,
		logger:       logger,
		cooldownChan: cooldownChan,
	}

	go instance.listenOrders()

	return instance
}

func (p *ProcessingController) listenOrders() {
	ticker := time.NewTicker(chanLen * time.Second)
	defer ticker.Stop()

	for {
		select {
		case cooldown := <-p.cooldownChan:
			time.Sleep(cooldown)
			continue
		case <-ticker.C:
		}
		orders, err := p.store.GetUnprocessedOrders()
		if err != nil {
			p.logger.Errorf("error getting unprocessed orders from store: %v", err)
		}
		for i := range orders {
			p.ordersChan <- &orders[i]
		}
	}
}

func (p *ProcessingController) Process(ctx context.Context) {
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case o := <-p.ordersChan:

				if o.Status == models.NEW {
					_, err := p.store.UpdateOrder(&models.Order{Number: o.Number, Status: models.PROCESSING})
					if err != nil {
						p.logger.Errorf("error updating order from accrual: %w", err)
						continue
					}
				}

				info, err := p.accrual.GetOrderInfo(o.Number)
				if err != nil {
					var serviceBusyError *accrual.ServiceBusyError
					if errors.As(err, &serviceBusyError) {
						p.logger.Infof("service busy: %v", serviceBusyError)
						p.cooldownChan <- serviceBusyError.CoolDown
						time.Sleep(serviceBusyError.CoolDown)
						continue
					}
					p.logger.Errorf("unhandled error: %v", err)
					continue
				}

				if info.Status == models.PROCESSED || info.Status == models.INVALID {
					_, err := p.store.UpdateOrder(
						&models.Order{
							Number:  info.Order,
							UserID:  o.UserID,
							Accrual: info.Accrual,
							Status:  info.Status,
						})
					if err != nil {
						p.logger.Errorf("error updating order: %w", err)
					}
				}
			}
		}
	}(ctx)
}
