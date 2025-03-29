package updatebalance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"wallets/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Мок репозитория
type mockBalanceUpdater struct {
	mock.Mock
}

func (m *mockBalanceUpdater) UpdateBalance(ctx context.Context, walletID uuid.UUID, operationType models.OperationType, amount int64) (models.Transactions, error) {
	args := m.Called(ctx, walletID, operationType, amount)
	return args.Get(0).(models.Transactions), args.Error(1)
}

func TestUpdateBalance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUUID, _ := uuid.NewV4()
	transactionUUID, _ := uuid.NewV4()

	tests := []struct {
		name           string
		body           Request
		mockTx         models.Transactions
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success",
			body: Request{
				ID:        validUUID,
				Operation: models.DEPOSIT,
				Amount:    1000,
			},
			mockTx: models.Transactions{
				ID:            transactionUUID,
				WalletID:      validUUID,
				OperationType: models.DEPOSIT,
				Amount:        1000,
			},
			mockError:      nil,
			expectedStatus: http.StatusAccepted,
			expectedBody:   transactionUUID.String(),
		},
		{
			name: "Invalid UUID",
			body: Request{
				ID:        uuid.UUID{},
				Operation: models.DEPOSIT,
				Amount:    1000,
			},
			mockTx:         models.Transactions{},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "ID is required", // TODO move to const errtranslate.go
		},
		{
			name: "Invalid opration",
			body: Request{
				ID:        validUUID,
				Operation: "INVALID",
				Amount:    1000,
			},
			mockTx:         models.Transactions{},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Operation must be in (DEPOSIT WITHDRAW)", // TODO move to const errtranslate.go
		},
		{
			name: "amount less",
			body: Request{
				ID:        validUUID,
				Operation: models.DEPOSIT,
			},
			mockTx:         models.Transactions{},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Amount is required", // TODO move to const errtranslate.go
		},
		{
			name: "amount < 0",
			body: Request{
				ID:        validUUID,
				Operation: models.DEPOSIT,
				Amount:    -3,
			},
			mockTx:         models.Transactions{},
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Amount must be greater than or equal to 1", // TODO move to const errtranslate.go
		},
		{
			name: "repo update balance error",
			body: Request{
				ID:        validUUID,
				Operation: models.WITHDRAW,
				Amount:    500,
			},
			mockTx:         models.Transactions{},
			mockError:      errors.New("db error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to update balance",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			gin.SetMode(gin.TestMode)

			log := slog.New(slog.DiscardHandler)
			mockRepo := new(mockBalanceUpdater)
			mockRepo.On("UpdateBalance", mock.Anything, tc.body.ID, tc.body.Operation, tc.body.Amount).Return(tc.mockTx, tc.mockError)

			reqBody, _ := json.Marshal(tc.body)
			req, _ := http.NewRequest("POST", "/wallet", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r := gin.Default()
			r.POST("/wallet", New(context.Background(), log, mockRepo))
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}
