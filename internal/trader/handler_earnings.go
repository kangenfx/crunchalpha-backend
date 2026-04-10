package trader

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type EarningsSummary struct {
	PendingEarnings  float64            `json:"pending_earnings"`
	InvoicedAmount   float64            `json:"invoiced_amount"`
	ReceivedAmount   float64            `json:"received_amount"`
	TotalEarned      float64            `json:"total_earned"`
	PerInvestor      []InvestorEarnings `json:"per_investor"`
}

type InvestorEarnings struct {
	InvestorName    string  `json:"investor_name"`
	TraderAccount   string  `json:"trader_account"`
	AumUSD          float64 `json:"aum_usd"`
	AccruedFee      float64 `json:"accrued_fee"`
	PaidFee         float64 `json:"paid_fee"`
	Status          string  `json:"status"`
}

func (h *Handler) GetEarnings(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var summary EarningsSummary
	summary.PerInvestor = []InvestorEarnings{}

	// Pending earnings (trader_share dari transaksi pending)
	h.service.repo.db.QueryRow(`
		SELECT COALESCE(SUM(ft.trader_share), 0)
		FROM investor_fee_transactions ft
		JOIN trader_accounts ta ON ta.id = ft.trader_account_id
		WHERE ta.user_id = $1::uuid
		AND ft.status = 'pending'
	`, userID).Scan(&summary.PendingEarnings)

	// Invoiced amount
	h.service.repo.db.QueryRow(`
		SELECT COALESCE(SUM(ft.trader_share), 0)
		FROM investor_fee_transactions ft
		JOIN trader_accounts ta ON ta.id = ft.trader_account_id
		WHERE ta.user_id = $1::uuid
		AND ft.status = 'invoiced'
	`, userID).Scan(&summary.InvoicedAmount)

	// Received amount
	h.service.repo.db.QueryRow(`
		SELECT COALESCE(SUM(ft.trader_share), 0)
		FROM investor_fee_transactions ft
		JOIN trader_accounts ta ON ta.id = ft.trader_account_id
		WHERE ta.user_id = $1::uuid
		AND ft.status = 'paid'
	`, userID).Scan(&summary.ReceivedAmount)

	summary.TotalEarned = summary.InvoicedAmount + summary.ReceivedAmount

	// Per-investor breakdown
	rows, err := h.service.repo.db.Query(`
		SELECT
			COALESCE(u.name, u.email) as investor_name,
			COALESCE(ta.nickname, ta.account_number) as trader_account,
			COALESCE(fp.aum_usd, 0) as aum_usd,
			COALESCE(fp.fee_amount, 0) as accrued_fee,
			COALESCE(ft.trader_share, 0) as paid_fee,
			COALESCE(ft.status, 'pending') as status
		FROM investor_fee_periods fp
		JOIN trader_accounts ta ON ta.id = fp.trader_account_id
		JOIN users u ON u.id = fp.user_id
		LEFT JOIN investor_fee_transactions ft ON ft.fee_period_id = fp.id
		WHERE ta.user_id = $1::uuid
		ORDER BY fp.created_at DESC
		LIMIT 50
	`, userID)

	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"ok": true, "data": summary})
		return
	}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var inv InvestorEarnings
			rows.Scan(&inv.InvestorName, &inv.TraderAccount, &inv.AumUSD, &inv.AccruedFee, &inv.PaidFee, &inv.Status)
			summary.PerInvestor = append(summary.PerInvestor, inv)
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "data": summary})
}
