package getbalance

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"wallets/internal/herrors"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testCase struct {
	name              string
	walletID          string
	mockBalanceWallet int64
	mockError         error
	expectedStatus    int
	expectedBody      string
}

type mockBalanceWallet struct {
	mock.Mock
}

func (m *mockBalanceWallet) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).(int64), args.Error(1)
}

func TestNew(t *testing.T) {

	gin.SetMode(gin.TestMode)

	validUUID, _ := uuid.NewV4()

	tests := []testCase{
		{
			name:              "Success test",
			walletID:          validUUID.String(),
			mockBalanceWallet: 5000,
			mockError:         nil,
			expectedStatus:    http.StatusAccepted,
			expectedBody:      `"balance":5000`,
		},
		{
			name:              "Incorrect UUID",
			walletID:          "I-n-c-o-r-r-e-c-t-uuid",
			mockBalanceWallet: 0,
			mockError:         nil, //TODO распарсить ошибки валидатора
			expectedStatus:    http.StatusBadRequest,
			expectedBody:      "failed to decode request",
		},
		{
			name:              "wallet not found",
			walletID:          validUUID.String(),
			mockBalanceWallet: 0,
			mockError:         herrors.ErrNXUUID,
			expectedStatus:    http.StatusInternalServerError,
			expectedBody:      "failed to get balance",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(mockBalanceWallet)

			log := slog.New(slog.DiscardHandler)

			if tc.walletID == validUUID.String() {
				mockRepo.On("GetBalance", mock.Anything, validUUID).Return(tc.mockBalanceWallet, tc.mockError)
			}

			req, _ := http.NewRequest("GET", "/wallets/"+tc.walletID, nil)
			w := httptest.NewRecorder()

			r := gin.Default()
			r.GET("/wallets/:uuid", New(context.Background(), log, mockRepo))
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.expectedBody)
		})
	}
}
