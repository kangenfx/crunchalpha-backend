package alpharank

import (
	"encoding/json"
	"net/http"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CalculateAlphaRank(c *gin.Context) {
	accountID := c.Param("account_id")
	
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}
	
	err := h.service.CalculateForAccount(accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"ok": true,
		"message": "AlphaRank calculated successfully",
		"account_id": accountID,
	})
}

// Add the detailed method to existing handler

// GET /api/public/traders — top traders by alpharank, no auth
func (h *Handler) GetPublicTraders(c *gin.Context) {
	rows, err := h.service.db.Query(`
		SELECT ta.id::text, COALESCE(ta.nickname, ta.account_number) as nickname,
		       COALESCE(ta.broker,'') as broker, COALESCE(ta.platform::text,'') as platform,
		       COALESCE(ar.alpha_score,0), COALESCE(ar.grade,'D'),
		       COALESCE(ar.win_rate,0), COALESCE(ar.max_drawdown_pct,0),
		       COALESCE(ar.roi,0), COALESCE(ar.net_pnl,0),
		       COALESCE(ar.total_trades_all,0), COALESCE(ar.profit_factor,0),
		       COALESCE(u.name, ta.nickname, ta.account_number) as trader_name,
		       COALESCE(ar.risk_level,'MEDIUM') as risk_level,
		       COALESCE(ta.about,'') as about
		FROM alpha_ranks ar
		JOIN trader_accounts ta ON ta.id = ar.account_id
		LEFT JOIN users u ON u.id = ta.user_id
		WHERE ar.symbol='ALL' AND ar.alpha_score > 0 AND ta.status='active'
		ORDER BY ar.alpha_score DESC
		LIMIT 10`)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer rows.Close()

	type TraderRow struct {
		ID           string  `json:"id"`
		Nickname     string  `json:"nickname"`
		TraderName   string  `json:"traderName"`
		Broker       string  `json:"broker"`
		Platform     string  `json:"platform"`
		AlphaScore   float64 `json:"alphaScore"`
		Grade        string  `json:"grade"`
		WinRate      float64 `json:"winRate"`
		MaxDD        float64 `json:"maxDD"`
		ROI          float64 `json:"roi"`
		NetPnl       float64 `json:"netPnl"`
		TotalTrades  int     `json:"totalTrades"`
		ProfitFactor float64 `json:"profitFactor"`
		RiskLevel   string  `json:"riskLevel"`
		Strategy    string  `json:"strategy"`
	}
	var traders []TraderRow
	for rows.Next() {
		var t TraderRow
		rows.Scan(&t.ID, &t.Nickname, &t.Broker, &t.Platform,
				&t.AlphaScore, &t.Grade, &t.WinRate, &t.MaxDD,
				&t.ROI, &t.NetPnl, &t.TotalTrades, &t.ProfitFactor,
				&t.TraderName, &t.RiskLevel, &t.Strategy)
		traders = append(traders, t)
	}
	if traders == nil { traders = []TraderRow{} }
	c.JSON(200, gin.H{"ok":true,"traders":traders,"count":len(traders)})
}

// GET /api/public/trader/:id — public trader detail with per pair and flags
func (h *Handler) GetPublicTraderDetail(c *gin.Context) {
	accountID := c.Param("id")

	// Per pair
	pairRows, err := h.service.db.Query(`
		SELECT ar.symbol, ar.trade_count, ar.win_rate, ar.profit_factor,
		       COALESCE(ar.net_pnl,0), COALESCE(ar.max_drawdown_pct,0), ar.grade, ar.alpha_score,
		       COALESCE(ar.risk_flags,'[]'::jsonb), COALESCE(ar.pillars,'[]'::jsonb)
		FROM alpha_ranks ar
		WHERE account_id=$1::uuid AND ar.symbol != 'ALL'
		ORDER BY ar.alpha_score DESC LIMIT 8`, accountID)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer pairRows.Close()

	type PairRow struct {
		Symbol       string  `json:"symbol"`
		Trades       int     `json:"trades"`
		WinRate      float64 `json:"winRate"`
		ProfitFactor float64 `json:"profitFactor"`
		RiskLevel   string  `json:"riskLevel"`
		Strategy    string  `json:"strategy"`
		NetPnl       float64 `json:"netPnl"`
		MaxDD        float64 `json:"maxDD"`
		AlphaScore   float64 `json:"alphaScore"`
		Grade        string  `json:"grade"`
	}
	var pairs []PairRow
	for pairRows.Next() {
		var p PairRow
		var flagsJSON, pillarsJSON []byte
		pairRows.Scan(&p.Symbol, &p.Trades, &p.WinRate, &p.ProfitFactor,
			&p.NetPnl, &p.MaxDD, &p.Grade, &p.AlphaScore,
			&flagsJSON, &pillarsJSON)
		_ = flagsJSON
		_ = pillarsJSON
		pairs = append(pairs, p)
	}
	if pairs == nil { pairs = []PairRow{} }

	// ALL stats dari alpha_ranks - single source of truth
	var allStats struct {
		AlphaScore    float64
		Grade         string
		Tier          string
		WinRate       float64
		ProfitFactor  float64
		MaxDD         float64
		NetPnl        float64
		TotalTrades   int
		WinningTrades int
		LosingTrades  int
		AvgWin        float64
		AvgLoss       float64
		RiskReward    float64
		Expectancy    float64
		Roi           float64
		RiskLevel     string
		Survivability float64
		Scalability   float64
		FlagsJSON     []byte
		PillarsJSON   []byte
	}
	h.service.db.QueryRow(`
		SELECT alpha_score, grade, COALESCE(tier,''), win_rate, profit_factor,
		       max_drawdown_pct, COALESCE(net_pnl,0), COALESCE(total_trades_all,0),
		       COALESCE(winning_trades,0), COALESCE(losing_trades,0),
		       COALESCE(avg_win,0), COALESCE(avg_loss,0),
		       COALESCE(risk_reward,0), COALESCE(expectancy,0),
		       COALESCE(roi,0), COALESCE(risk_level,'MEDIUM'),
		       COALESCE(survivability_score,0), COALESCE(scalability_score,0),
		       COALESCE(risk_flags,'[]'::jsonb), COALESCE(pillars,'[]'::jsonb)
		FROM alpha_ranks WHERE account_id=$1::uuid AND symbol='ALL'
	`, accountID).Scan(
		&allStats.AlphaScore, &allStats.Grade, &allStats.Tier,
		&allStats.WinRate, &allStats.ProfitFactor, &allStats.MaxDD,
		&allStats.NetPnl, &allStats.TotalTrades,
		&allStats.WinningTrades, &allStats.LosingTrades,
		&allStats.AvgWin, &allStats.AvgLoss,
		&allStats.RiskReward, &allStats.Expectancy,
		&allStats.Roi, &allStats.RiskLevel,
		&allStats.Survivability, &allStats.Scalability,
		&allStats.FlagsJSON, &allStats.PillarsJSON,
	)

	// Parse flags - tampilkan
	var flags []map[string]interface{}
	if err := json.Unmarshal(allStats.FlagsJSON, &flags); err != nil || flags == nil {
		flags = []map[string]interface{}{}
	}

	// Parse pillars - SEMBUNYIKAN weight dan reason
	type PillarPublic struct {
		Code  string  `json:"code"`
		Name  string  `json:"name"`
		Score float64 `json:"score"`
	}
	var pillarsRaw []struct {
		Code   string  `json:"code"`
		Name   string  `json:"name"`
		Score  float64 `json:"score"`
	}
	json.Unmarshal(allStats.PillarsJSON, &pillarsRaw)
	pillarsPublic := []PillarPublic{}
	for _, p := range pillarsRaw {
		pillarsPublic = append(pillarsPublic, PillarPublic{Code: p.Code, Name: p.Name, Score: p.Score})
	}

	c.JSON(200, gin.H{
		"ok": true,
		"alphaScore":    allStats.AlphaScore,
		"grade":         allStats.Grade,
		"tier":          allStats.Tier,
		"winRate":       allStats.WinRate,
		"profitFactor":  allStats.ProfitFactor,
		"maxDD":         allStats.MaxDD,
		"netPnl":        allStats.NetPnl,
		"totalTrades":   allStats.TotalTrades,
		"winningTrades": allStats.WinningTrades,
		"losingTrades":  allStats.LosingTrades,
		"avgWin":        allStats.AvgWin,
		"avgLoss":       allStats.AvgLoss,
		"riskReward":    allStats.RiskReward,
		"expectancy":    allStats.Expectancy,
		"roi":           allStats.Roi,
		"riskLevel":     allStats.RiskLevel,
		"survivability": allStats.Survivability,
		"scalability":   allStats.Scalability,
		"flags":         flags,
		"pillars":       pillarsPublic,
		"pairs":         pairs,
	})
}
