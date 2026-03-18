package alpharank

import (
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
		       COALESCE(u.name, ta.nickname, ta.account_number) as trader_name
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
	}
	var traders []TraderRow
	for rows.Next() {
		var t TraderRow
		rows.Scan(&t.ID, &t.Nickname, &t.Broker, &t.Platform,
			&t.AlphaScore, &t.Grade, &t.WinRate, &t.MaxDD,
			&t.ROI, &t.NetPnl, &t.TotalTrades, &t.ProfitFactor, &t.TraderName)
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
		SELECT symbol, trade_count as total_trades, win_rate, profit_factor,
		       COALESCE(risk_reward,0) as avg_rr, grade
		FROM alpha_ranks
		WHERE account_id=$1::uuid AND symbol != 'ALL'
		ORDER BY alpha_score DESC LIMIT 8`, accountID)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer pairRows.Close()

	type PairRow struct {
		Symbol       string  `json:"symbol"`
		Trades       int     `json:"trades"`
		WinRate      float64 `json:"winRate"`
		ProfitFactor float64 `json:"profitFactor"`
		AvgRR        float64 `json:"avgRR"`
		Grade        string  `json:"grade"`
	}
	var pairs []PairRow
	for pairRows.Next() {
		var p PairRow
		pairRows.Scan(&p.Symbol, &p.Trades, &p.WinRate, &p.ProfitFactor, &p.AvgRR, &p.Grade)
		pairs = append(pairs, p)
	}
	if pairs == nil { pairs = []PairRow{} }

	// Risk flags
	var flagsJSON string
	h.service.db.QueryRow(`SELECT COALESCE(risk_flags::text,'[]') FROM alpha_ranks
		WHERE account_id=$1::uuid AND symbol='ALL'`, accountID).Scan(&flagsJSON)

	c.JSON(200, gin.H{"ok":true,"pairs":pairs,"flagsJson":flagsJSON})
}
