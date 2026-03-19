package trader

import (
	"encoding/json"
	"net/http"

	"crunchalpha-v3/internal/alpharank"
	"github.com/gin-gonic/gin"
)

// GetAlphaRankPerPair - reads ALL data from DB, no on-the-fly calculation
func (h *Handler) GetAlphaRankPerPair(c *gin.Context) {
	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	account, err := h.service.repo.GetAccountByID(accountID, userID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	// Read ALL per-pair data from DB - single source of truth
	rows, err := h.service.repo.db.Query(`
		SELECT
			ar.symbol,
			ar.alpha_score,
			ar.grade,
			ar.badge,
			ar.tier,
			ar.max_drawdown_pct,
			COALESCE(ar.risk_flags, '[]'::jsonb),
			ar.critical_count,
			ar.major_count,
			ar.minor_count,
			COALESCE(ar.pillars, '[]'::jsonb),
			ar.trade_count,
			COALESCE(t.wins, 0),
			COALESCE(t.net_profit, 0)
		FROM alpha_ranks ar
		LEFT JOIN (
			SELECT
				account_id,
				symbol,
				SUM(CASE WHEN profit > 0 THEN 1 ELSE 0 END) as wins,
				SUM(profit) as net_profit
			FROM trades
			WHERE account_id = $1 AND status = 'closed'
			GROUP BY account_id, symbol
		) t ON t.symbol = ar.symbol AND t.account_id = ar.account_id
		WHERE ar.account_id = $1 AND ar.symbol != 'ALL'
		ORDER BY ar.symbol
	`, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read from db"})
		return
	}
	defer rows.Close()

	pairResults := []map[string]interface{}{}

	for rows.Next() {
		var symbol, grade, badge, tier string
		var score, maxDD, netProfit float64
		var critical, major, minor, tradeCount, wins int
		var flagsJSON, pillarsJSON []byte

		err := rows.Scan(
			&symbol, &score, &grade, &badge, &tier, &maxDD,
			&flagsJSON, &critical, &major, &minor,
			&pillarsJSON, &tradeCount, &wins, &netProfit,
		)
		if err != nil {
			continue
		}

		// Parse flags from DB
		var flags []alpharank.RiskFlag
		json.Unmarshal(flagsJSON, &flags)
		flagsResponse := []map[string]interface{}{}
		for _, flag := range flags {
			flagsResponse = append(flagsResponse, map[string]interface{}{
				"name":     flag.Title,
				"severity": flag.Severity,
				"desc":     flag.Desc,
			})
		}

		// Parse pillars from DB
		var pillars []alpharank.PillarScore
		json.Unmarshal(pillarsJSON, &pillars)
		pillarsResponse := []map[string]interface{}{}
		for _, pillar := range pillars {
			pillarsResponse = append(pillarsResponse, map[string]interface{}{
				"code":   pillar.Code,
				"name":   pillar.Name,
				"score":  pillar.Score,
				"weight": pillar.Weight,
			})
		}

		winRate := 0.0
		if tradeCount > 0 {
			winRate = float64(wins) / float64(tradeCount) * 100
		}

		pairResults = append(pairResults, map[string]interface{}{
			"symbol":     symbol,
			"score":      score,
			"grade":      grade,
			"tier":       badge,
			"trades":     tradeCount,
			"win_rate":   winRate,
			"net_profit": netProfit,
			"wins":       wins,
			"losses":     tradeCount - wins,
			"max_dd":     maxDD,
			"pillars":    pillarsResponse,
			"flags":      flagsResponse,
		})
	}

	if len(pairResults) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"account_id":     accountID,
			"account_number": account.AccountNumber,
			"pairs":          []map[string]interface{}{},
			"total_symbols":  0,
			"message":        "No per-pair data. Run recalculate first.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id":     accountID,
		"account_number": account.AccountNumber,
		"pairs":          pairResults,
		"total_symbols":  len(pairResults),
	})
}
