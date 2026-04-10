package alpharank

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetDetailedAlphaRank(c *gin.Context) {
	accountID := c.Param("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}

	var r struct {
		AlphaScore    float64
		Grade         string
		Badge         string
		TradeCount    int
		SurvScore     int
		ScalScore     int
		RiskLevel     string
		MaxDD         float64
		WinRate       float64
		NetPnl        float64
		Roi           float64
		ProfitFactor  float64
		TotalTrades   int
		WinTrades     int
		LoseTrades    int
		AvgWin        float64
		AvgLoss       float64
		RiskReward    float64
		Expectancy    float64
		TotalDeposit  float64
		TotalWithdraw float64
		P1            float64
		P2            float64
		P3            float64
		P4            float64
		P5            float64
		P6            float64
		P7            float64
		FlagsJSON     []byte
		PillarsJSON   []byte
		L3Multiplier  float64
		L3Status      string
		L3Reason      string
		L3Detail      []byte
	}

	err := h.service.db.QueryRow(`
		SELECT
			alpha_score, grade, COALESCE(badge,''), trade_count,
			COALESCE(survivability_score,0), COALESCE(scalability_score,0),
			COALESCE(risk_level,'MEDIUM'), COALESCE(max_drawdown_pct,0),
			COALESCE(win_rate,0), COALESCE(net_pnl,0), COALESCE(roi,0),
			COALESCE(profit_factor,0), COALESCE(total_trades_all,0),
			COALESCE(winning_trades,0), COALESCE(losing_trades,0),
			COALESCE(avg_win,0), COALESCE(avg_loss,0),
			COALESCE(risk_reward,0), COALESCE(expectancy,0),
			COALESCE(total_deposit,0), COALESCE(total_withdraw,0),
			COALESCE(profitability_score,0), COALESCE(consistency_score,0),
			COALESCE(risk_score,0), COALESCE(stability_score,0),
			COALESCE(activity_score,0), COALESCE(duration_score,0),
			COALESCE(drawdown_score,0),
			COALESCE(risk_flags,'[]'::jsonb),
			COALESCE(pillars,'[]'::jsonb),
			COALESCE(layer3_multiplier,1.0),
			COALESCE(layer3_status,'NEUTRAL'),
			COALESCE(layer3_reason,''),
			COALESCE(layer3_detail,'{}'::jsonb)
		FROM alpha_ranks
		WHERE account_id = $1 AND symbol = 'ALL'
	`, accountID).Scan(
		&r.AlphaScore, &r.Grade, &r.Badge, &r.TradeCount,
		&r.SurvScore, &r.ScalScore,
		&r.RiskLevel, &r.MaxDD,
		&r.WinRate, &r.NetPnl, &r.Roi,
		&r.ProfitFactor, &r.TotalTrades,
		&r.WinTrades, &r.LoseTrades,
		&r.AvgWin, &r.AvgLoss,
		&r.RiskReward, &r.Expectancy,
		&r.TotalDeposit, &r.TotalWithdraw,
		&r.P1, &r.P2, &r.P3, &r.P4, &r.P5, &r.P6, &r.P7,
		&r.FlagsJSON, &r.PillarsJSON,
		&r.L3Multiplier, &r.L3Status, &r.L3Reason, &r.L3Detail,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "AlphaRank not found: " + err.Error()})
		return
	}

	// Flags dari DB — soft_title/soft_desc sudah ada di JSONB
	var flags []map[string]interface{}
	if err := json.Unmarshal(r.FlagsJSON, &flags); err != nil || flags == nil {
		flags = []map[string]interface{}{}
	}

	// Pillars dari DB
	type PillarPublic struct {
		Code  string  `json:"code"`
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	}
	var pillarsRaw []PillarPublic
	json.Unmarshal(r.PillarsJSON, &pillarsRaw)
	if pillarsRaw == nil {
		pillarsRaw = []PillarPublic{}
	}

	// Flag counts dari JSONB
	flagCounts := struct {
		Critical int `json:"critical"`
		Major    int `json:"major"`
		Minor    int `json:"minor"`
	}{}
	for _, flag := range flags {
		switch flag["severity"] {
		case "CRITICAL":
			flagCounts.Critical++
		case "MAJOR":
			flagCounts.Major++
		case "MINOR":
			flagCounts.Minor++
		}
	}

	// Layer3 detail
	var l3Detail map[string]interface{}
	json.Unmarshal(r.L3Detail, &l3Detail)
	if l3Detail == nil {
		l3Detail = map[string]interface{}{}
	}

	c.JSON(http.StatusOK, gin.H{
		"account_id": accountID,
		"alpharank": gin.H{
			"score":      r.AlphaScore,
			"grade":      r.Grade,
			"tier":       r.Badge,
			"risk_level": r.RiskLevel,
		},
		"pillars": gin.H{
			"P1_Profitability":  r.P1,
			"P2_Consistency":    r.P2,
			"P3_RiskManagement": r.P3,
			"P4_Recovery":       r.P4,
			"P5_Edge":           r.P5,
			"P6_Discipline":     r.P6,
			"P7_TrackRecord":    r.P7,
			"detail":            pillarsRaw,
		},
		"risk_flags": gin.H{
			"total":  len(flags),
			"counts": flagCounts,
			"items":  flags,
		},
		"survivability": gin.H{
			"score": r.SurvScore,
			"label": survLabel(r.SurvScore),
		},
		"scalability": gin.H{
			"score": r.ScalScore,
			"label": survLabel(r.ScalScore),
		},
		"metrics": gin.H{
			"total_trades":      r.TotalTrades,
			"winning_trades":    r.WinTrades,
			"losing_trades":     r.LoseTrades,
			"win_rate":          r.WinRate,
			"net_pnl":           r.NetPnl,
			"roi":               r.Roi,
			"profit_factor":     r.ProfitFactor,
			"avg_win":           r.AvgWin,
			"avg_loss":          r.AvgLoss,
			"risk_reward":       r.RiskReward,
			"expectancy":        r.Expectancy,
			"max_drawdown_pct":  r.MaxDD,
			"total_deposits":    r.TotalDeposit,
			"total_withdrawals": r.TotalWithdraw,
		},
		"layer3": gin.H{
			"multiplier": r.L3Multiplier,
			"status":     r.L3Status,
			"reason":     r.L3Reason,
			"detail":     l3Detail,
		},
	})
}

func survLabel(score int) string {
	switch {
	case score >= 80:
		return "Excellent"
	case score >= 60:
		return "Good"
	case score >= 40:
		return "Moderate"
	default:
		return "Poor"
	}
}
