package create

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	resp "wallets/internal/http-server/api/response"
	"wallets/internal/lib/sl"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type Request struct {
	Balance int64 `json:"balance" binding:"required"`
}

type Response struct {
	resp.Response
	ID uuid.UUID `json:"id"`
}

type walletCreator interface {
	CreateWallet(ctx context.Context, balance int64) (uuid.UUID, error)
}

func New(ctx context.Context, log *slog.Logger, repos walletCreator) gin.HandlerFunc {

	return func(c *gin.Context) {
		const op = "handlers.wallets.create.New"

		log := log.With(slog.String("op", op))

		var req Request

		err := c.ShouldBindJSON(&req)

		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Info("empty request body, using default balance", slog.Int("balance", 0))
				req.Balance = 0
			} else {
				log.Error("failed to decode request body", sl.Err(err))
				c.JSON(http.StatusBadRequest, resp.Error("failed to decode request"))
				return
			}
		}

		log.Info("request body decoded", slog.Any("request", req))

		id, err := repos.CreateWallet(ctx, req.Balance)
		if err != nil {
			log.Error("failed to create wallet", sl.Err(err))

			c.JSON(http.StatusInternalServerError, resp.Error("failed to create wallet"))

			return
		}

		c.JSON(http.StatusCreated, Response{
			Response: resp.OK(),
			ID:       id,
		})

	}
}
