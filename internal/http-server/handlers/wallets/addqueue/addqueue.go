package addqueue

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	resp "wallets/internal/http-server/api/response"
	"wallets/internal/lib/errtranslate"
	"wallets/internal/lib/sl"
	"wallets/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
)

type Request struct {
	ID        uuid.UUID            `json:"wallet_id" binding:"required,uuid4"`
	Operation models.OperationType `json:"operation_type" binding:"required,oneof=DEPOSIT WITHDRAW"`
	Amount    int64                `json:"amount" binding:"required,gte=1"`
}

type Response struct {
	resp.Response
	TxID string `json:"tx_id"`
}

type QueueManager interface {
	AddToQueue(ctx context.Context, walletID uuid.UUID, operation string, amount int64) (string, error)
}

func New(ctx context.Context, log *slog.Logger, repos QueueManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handlers.wallets.addqueue.New"
		log := log.With("op", op)
		var req Request

		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error("failed to decode request", sl.Err(err))

			msg := "failed to decode request"

			if validationErrs, ok := err.(validator.ValidationErrors); ok {
				fieldErrors := errtranslate.TranslateValidationErrors(validationErrs)
				msg = strings.Join(fieldErrors, ", ")
			}

			c.JSON(http.StatusBadRequest, resp.Error(msg))
			return
		}

		txID, err := repos.AddToQueue(ctx, req.ID, string(req.Operation), req.Amount)
		if err != nil {
			log.Error("failed to add transaction to queue", sl.Err(err))
			c.JSON(http.StatusInternalServerError, resp.Error("failed to add transaction"))
			return
		}

		log.Info("Transaction added to queue", slog.String("TransactionID", txID))
		c.JSON(http.StatusAccepted, Response{
			Response: resp.OK(),
			TxID:     txID,
		})

	}
}
