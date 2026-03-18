package alpharank

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetDetailedAlphaRank(c *gin.Context) {
	accountID := c.Param("account_id")
	
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}
	
	var result struct {
		P1, P2, P3, P4, P5, P6, P7 float64
		AlphaScore float64
		Grade string
		Badge string
		TradeCount int
	}
	
	err := h.service.db.QueryRow(`
		SELECT 
			profitability_score, risk_score, consistency_score,
			stability_score, activity_score, duration_score, drawdown_score,
			alpha_score, grade, badge, trade_count
		FROM alpha_ranks
		WHERE account_id = $1 AND symbol = 'ALL'
	`, accountID).Scan(
		&result.P1, &result.P2, &result.P3, &result.P4, 
		&result.P5, &result.P6, &result.P7,
		&result.AlphaScore, &result.Grade, &result.Badge, &result.TradeCount,
	)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "AlphaRank not found"})
		return
	}
	
	trades, err := h.service.getTradesForAccount(accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get trades"})
		return
	}
	
	var balance, equity float64
	h.service.db.QueryRow(`
		SELECT COALESCE(balance, 0), COALESCE(equity, 0)
		FROM trader_accounts WHERE id = $1
	`, accountID).Scan(&balance, &equity)
	var totalDeposits, totalWithdrawals float64
	h.service.db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM account_transactions WHERE account_id = $1 AND transaction_type = 'deposit'", accountID).Scan(&totalDeposits)
	h.service.db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM account_transactions WHERE account_id = $1 AND transaction_type = 'withdrawal'", accountID).Scan(&totalWithdrawals)
	metrics := h.service.buildMetrics(accountID, trades, balance, equity, totalDeposits, totalWithdrawals)

	// Use maxDD from DB (same source as stats bar) for consistent flag descriptions
	var dbMaxDD float64
	h.service.db.QueryRow(`
		SELECT COALESCE(max_dd, 0) FROM alpha_ranks
		WHERE account_id = $1 AND symbol = 'ALL'
	`, accountID).Scan(&dbMaxDD)
	if dbMaxDD > 0 {
		metrics.MaxDrawdownPct = dbMaxDD
	}
	
	flags := DetectRiskFlags(metrics)
	
	calculator := NewCalculator()
	surv := calculator.CalculateSurvivability(metrics.MaxDrawdownPct, result.AlphaScore)
	scal := calculator.CalculateScalability(metrics.PeakBalance, result.AlphaScore)
	
	// REGIME DETECTION
	
	flagCounts := struct {
		Critical int `json:"critical"`
		Major    int `json:"major"`
		Minor    int `json:"minor"`
	}{}
	
	for _, flag := range flags {
		switch flag.Severity {
		case "CRITICAL":
			flagCounts.Critical++
		case "MAJOR":
			flagCounts.Major++
		case "MINOR":
			flagCounts.Minor++
		}
	}
	
	response := gin.H{
		"account_id": accountID,
		"alpharank": gin.H{
			"score": result.AlphaScore,
			"grade": result.Grade,
			"tier":  result.Badge,
		},
		"pillars": gin.H{
			"P1_Profitability": result.P1,
			"P2_Risk":          result.P2,
			"P3_Consistency":   result.P3,
			"P4_Recovery":      result.P4,
			"P5_Edge":          result.P5,
			"P6_Discipline":    result.P6,
			"P7_TrackRecord":   result.P7,
		},
		"risk_flags": gin.H{
			"total": len(flags),
			"counts": flagCounts,
			"items": flags,
		},
		"survivability": gin.H{
			"score": surv.Score,
			"label": surv.Label,
			"note":  surv.Note,
		},
		"scalability": gin.H{
			"score": scal.Score,
			"label": scal.Label,
			"note":  scal.Note,
		},
		"metrics": gin.H{
			"total_trades":     metrics.TotalTrades,
			"winning_trades":   metrics.WinningTrades,
			"losing_trades":    metrics.LosingTrades,
			"win_rate":         float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100,
				"initial_deposit":  metrics.InitialDeposit,
				"total_deposits":   totalDeposits,
				"total_withdrawals": totalWithdrawals,
			"net_profit":       metrics.NetProfit,
			"gross_profit":     metrics.GrossProfit,
			"gross_loss":       metrics.GrossLoss,
			"max_drawdown_pct": metrics.MaxDrawdownPct,
			"current_balance":  metrics.CurrentBalance,
			"current_equity":   metrics.CurrentEquity,
		},
	}
	
	c.JSON(http.StatusOK, response)
}
