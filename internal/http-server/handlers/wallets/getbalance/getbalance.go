package getbalance

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	resp "wallets/internal/http-server/api/response"
	"wallets/internal/lib/errtranslate"
	"wallets/internal/lib/sl"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
)

type Request struct {
	ID uuid.UUID `validate:"uuid4,required"`
}

type Response struct {
	resp.Response
	Balance int64 `json:"balance"`
}

type balanceWallet interface {
	GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error)
}

func New(ctx context.Context, log *slog.Logger, repos balanceWallet) gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "handlers.wallets.getbalance.New"

		log := log.With(slog.String("op", op))

		var req Request

		req_parm := uuid.UUID{}
		err := req_parm.Parse(c.Param("uuid"))
		if err != nil {
			log.Error("failed to decode request parametr", sl.Err(err))
			c.JSON(http.StatusBadRequest, resp.Error("failed to decode request"))
			return
		}

		req.ID = req_parm

		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			fieldErrors := errtranslate.TranslateValidationErrors(validationErrs)
			msg := strings.Join(fieldErrors, ", ")
			c.JSON(http.StatusBadRequest, resp.Error(msg))
			return
		}

		balance, err := repos.GetBalance(ctx, req.ID)
		if err != nil {
			log.Error("failed to get balance", sl.Err(err))
			c.JSON(http.StatusInternalServerError, resp.Error("failed to get balance"))
			return
		}

		c.JSON(http.StatusAccepted, Response{
			Response: resp.OK(),
			Balance:  balance,
		})

	}
}
