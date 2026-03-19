package ea

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

type TradeData struct {
	AccountNumber string  `json:"account_number" binding:"required"`
	Ticket        int64   `json:"ticket" binding:"required"`
	Symbol        string  `json:"symbol" binding:"required"`
	Type          string  `json:"type" binding:"required"`
	Lots          float64 `json:"lots" binding:"required"`
	OpenPrice     float64 `json:"open_price"`
	ClosePrice    float64 `json:"close_price,omitempty"`
	Profit        float64 `json:"profit,omitempty"`
	Swap          float64 `json:"swap,omitempty"`
	Commission    float64 `json:"commission,omitempty"`
	OpenTime      int64   `json:"open_time,omitempty"`
	CloseTime     int64   `json:"close_time,omitempty"`
	Timestamp     int64   `json:"timestamp" binding:"required"`
	Status        string  `json:"status" binding:"required"`
}

type AccountData struct {
	AccountNumber string  `json:"account_number" binding:"required"`
	Balance       float64 `json:"balance" binding:"required"`
	Equity        float64 `json:"equity" binding:"required"`
	Margin        float64 `json:"margin"`
	FreeMargin    float64 `json:"free_margin"`
	Timestamp     int64   `json:"timestamp" binding:"required"`
}

func (h *Handler) ReceiveTrade(c *gin.Context) {
	var data TradeData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	accountID, err := h.repo.GetAccountIDByNumber(data.AccountNumber, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	if err := h.repo.SaveTrade(accountID, &data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save trade"})
		return
	}
	// Trigger recalc saat trade closed
	if data.Status == "closed" {
		go h.repo.TriggerAlphaRankCalculation(accountID)
	}
	c.JSON(http.StatusCreated, gin.H{
		"ok":      true,
		"message": "trade received",
		"ticket":  data.Ticket,
	})
}

func (h *Handler) ReceiveAccount(c *gin.Context) {
	var data AccountData

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accountID, err := h.repo.GetAccountIDByNumber(data.AccountNumber, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if err := h.repo.UpdateAccountBalance(accountID, data.Balance, data.Equity); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update account"})
		return
	}

	h.repo.SaveEquitySnapshot(accountID, data.Balance, data.Equity)

	// Trigger recalculate setiap EA push account data
	go h.repo.TriggerAlphaRankCalculation(accountID)

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "account updated"})
}

func (h *Handler) SyncAccount(c *gin.Context) {
	var payload struct {
		AccountNumber    string      `json:"account_number" binding:"required"`
		Balance          float64     `json:"balance" binding:"required"`
		Equity           float64     `json:"equity" binding:"required"`
		InitialDeposit   float64     `json:"initial_deposit,omitempty"`
		TotalDeposits    float64     `json:"total_deposits,omitempty"`
		TotalWithdrawals float64     `json:"total_withdrawals,omitempty"`
		ClosedTrades     []TradeData `json:"closed_trades"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	accountID, err := h.repo.GetAccountIDByNumber(payload.AccountNumber, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	h.repo.UpdateAccountBalance(accountID, payload.Balance, payload.Equity)
	h.repo.SaveEquitySnapshot(accountID, payload.Balance, payload.Equity)

	fmt.Printf("[DEPOSITS] Account: %s, Initial: %.2f, Total: %.2f, Withdrawals: %.2f\n",
		payload.AccountNumber, payload.InitialDeposit, payload.TotalDeposits, payload.TotalWithdrawals)

	if payload.InitialDeposit > 0 || payload.TotalDeposits > 0 || payload.TotalWithdrawals > 0 {
		h.repo.SaveAccountTransactions(accountID, payload.InitialDeposit, payload.TotalDeposits, payload.TotalWithdrawals)
	}

	tradesAdded := 0
	for _, trade := range payload.ClosedTrades {
		if err := h.repo.SyncTrade(accountID, &trade); err == nil {
			tradesAdded++
		}
	}

	// Trigger recalculate setiap sync
	go h.repo.TriggerAlphaRankCalculation(accountID)

	c.JSON(http.StatusOK, gin.H{
		"ok":            true,
		"message":       "sync completed",
		"trades_synced": tradesAdded,
	})
}
