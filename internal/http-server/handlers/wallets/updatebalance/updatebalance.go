package updatebalance

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"wallets/internal/herrors"
	resp "wallets/internal/http-server/api/response"
	"wallets/internal/lib/errtranslate"
	"wallets/internal/lib/sl"
	"wallets/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
)

type Response struct {
	resp.Response
	TxID uuid.UUID `json:"transaction"`
}

type Request struct {
	ID        uuid.UUID            `json:"wallet_id" binding:"required,uuid4"`
	Operation models.OperationType `json:"operation_type" binding:"required,oneof=DEPOSIT WITHDRAW"`
	Amount    int64                `json:"amount" binding:"required,gte=1"`
}

type BalanceUpdater interface {
	UpdateBalance(ctx context.Context, walletID uuid.UUID, operationType models.OperationType, amount int64) (models.Transactions, error)
}

func New(ctx context.Context, log *slog.Logger, repos BalanceUpdater) gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handlers.wallets.updatebalance.New"

		log := log.With("op", op)

		var req Request

		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error("failed to decode request", sl.Err(err))

			if validationErrs, ok := err.(validator.ValidationErrors); ok {
				fieldErrors := errtranslate.TranslateValidationErrors(validationErrs)
				msg := strings.Join(fieldErrors, ", ")
				c.JSON(http.StatusBadRequest, resp.Error(msg))
				return
			}

			c.JSON(http.StatusBadRequest, resp.Error("failed to decode request"))
			return
		}

		tx, err := repos.UpdateBalance(ctx, req.ID, req.Operation, req.Amount)
		if err != nil {
			log.Error("failed to update balance", sl.Err(err))

			if errors.Is(err, herrors.ErrNXUUID) {
				c.JSON(http.StatusBadRequest, resp.Error("failed to find uuid"))
				return
			}

			if errors.Is(err, herrors.ErrInsufficientFunds) {
				c.JSON(http.StatusBadRequest, resp.Error("failed to WITHDRAW: insufficient funds"))
				return
			}

			c.JSON(http.StatusInternalServerError, resp.Error("failed to update balance"))
			return
		}

		c.JSON(http.StatusAccepted, tx)

	}
}
