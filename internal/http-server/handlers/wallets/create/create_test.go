package create

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"wallets/internal/http-server/api/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/assert/v2"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockWalletCreator struct {
	mock.Mock
}

type testCase struct {
	name           string
	requestBody    string
	mockReturnID   uuid.UUID
	mockReturnErr  error
	expectedCode   int
	expectedResp   Response
	expectRepoCall bool
}

func (m *mockWalletCreator) CreateWallet(ctx context.Context, balance int64) (uuid.UUID, error) {
	args := m.Called(ctx, balance)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func TestNew(t *testing.T) {

	gin.SetMode(gin.TestMode)

	walletID, _ := uuid.NewV4()

	testCases := []testCase{
		{
			name:          "Success",
			requestBody:   `{"balance": 1000}`,
			mockReturnID:  walletID,
			mockReturnErr: nil,
			expectedCode:  http.StatusCreated,
			expectedResp: Response{
				Response: response.OK(),
				ID:       walletID,
			},
			expectRepoCall: true,
		},

		{
			name:          "empty ballance",
			requestBody:   "",
			mockReturnID:  walletID,
			mockReturnErr: nil,
			expectedCode:  http.StatusCreated,
			expectedResp: Response{
				Response: response.OK(),
				ID:       walletID,
			},
			expectRepoCall: true,
		},

		{
			name:           "invalid request body",
			requestBody:    `{"invalid": "json"}`,
			expectedCode:   http.StatusBadRequest,
			expectRepoCall: false,
		},

		{
			name:           "incorrect balance numeric value",
			requestBody:    `{"balance": -132}`,
			expectedCode:   http.StatusInternalServerError,
			mockReturnErr:  errors.New("Repos error"),
			expectRepoCall: true,
		},

		{
			name:           "incorrect balance value",
			requestBody:    `{"balance":"asdasd"}`,
			expectedCode:   http.StatusBadRequest,
			mockReturnErr:  errors.New("balance must be positive integer"),
			expectRepoCall: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := new(mockWalletCreator)

			mockRepo.ExpectedCalls = nil
			if tc.expectRepoCall {
				mockRepo.On("CreateWallet", mock.Anything, mock.AnythingOfType("int64")).
					Return(tc.mockReturnID, tc.mockReturnErr).
					Once()
			}

			log := slog.New(slog.DiscardHandler)

			handler := New(context.Background(), log, mockRepo)

			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)

			c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/wallet/create", bytes.NewBufferString(tc.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			handler(c)

			assert.Equal(t, tc.expectedCode, w.Code)

			if tc.expectedCode == http.StatusCreated {
				var response Response
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tc.expectedResp.ID, response.ID)
			}

			mockRepo.AssertExpectations(t)
		})
	}

}
