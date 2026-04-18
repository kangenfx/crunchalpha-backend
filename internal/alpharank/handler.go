package alpharank

import (
	"encoding/json"
	"net/http"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"fmt"
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
	// Query params
	sortBy    := c.DefaultQuery("sort", "alpha_score")
	riskLevel := c.Query("risk")
	platform  := c.Query("platform")
	search    := c.Query("search")
	pageStr   := c.DefaultQuery("page", "1")
	limitStr  := c.DefaultQuery("limit", "12")

	page,  _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1  { page = 1 }
	if limit < 1 || limit > 50 { limit = 12 }
	offset := (page - 1) * limit

	// Sort whitelist
	sortCol := map[string]string{
		"alpha_score":   "ar.alpha_score DESC",
		"roi":           "ar.roi DESC",
		"win_rate":      "ar.win_rate DESC",
		"drawdown":      "ar.max_drawdown_pct ASC",
		"profit_factor": "ar.profit_factor DESC",
		"trades":        "ar.total_trades_all DESC",
	}
	orderBy, ok := sortCol[sortBy]
	if !ok { orderBy = "ar.alpha_score DESC" }

	// WHERE conditions
	where := `ar.symbol='ALL'
		AND ar.alpha_score > 0
		AND COALESCE(ar.total_trades_all,0) >= 10
		AND ta.status='active'
			AND ta.ea_verified = true`

	args := []interface{}{}
	argN := 1

	if riskLevel != "" && riskLevel != "ALL" {
		where += fmt.Sprintf(" AND ar.risk_level=$%d", argN)
		args = append(args, riskLevel)
		argN++
	}
	if platform != "" && platform != "ALL" {
		where += fmt.Sprintf(" AND ta.platform::text=$%d", argN)
		args = append(args, platform)
		argN++
	}
	if search != "" {
		where += fmt.Sprintf(" AND (LOWER(COALESCE(ta.nickname,ta.account_number)) LIKE $%d OR LOWER(ta.broker) LIKE $%d)", argN, argN)
		args = append(args, "%"+strings.ToLower(search)+"%")
		argN++
	}

	// Count total
	var total int
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM alpha_ranks ar
		JOIN trader_accounts ta ON ta.id = ar.account_id
		WHERE %s`, where)
	h.service.db.QueryRow(countSQL, args...).Scan(&total)

	// Main query
	queryArgs := append(args, limit, offset)
	sqlQ := fmt.Sprintf(`
		SELECT ta.id::text,
		       COALESCE(ta.nickname, ta.account_number) as nickname,
		       COALESCE(ta.broker,'') as broker,
		       COALESCE(ta.platform::text,'') as platform,
		       COALESCE(ar.alpha_score,0), COALESCE(ar.grade,'D'),
		       COALESCE(ar.win_rate,0), COALESCE(ar.max_drawdown_pct,0),
		       COALESCE(ar.roi,0), COALESCE(ar.net_pnl,0),
		       COALESCE(ar.total_trades_all,0), COALESCE(ar.profit_factor,0),
		       COALESCE(u.name, ta.nickname, ta.account_number) as trader_name,
		       COALESCE(ar.risk_level,'MEDIUM') as risk_level,
		       COALESCE(ta.about,'') as about,
                       COALESCE(ar.layer3_multiplier,1.0) as layer3_multiplier,
                       COALESCE(ar.layer3_status,'NEUTRAL') as layer3_status,
                       COALESCE(ar.layer3_reason,'') as layer3_reason,
				COALESCE(ta.currency,'USD') as currency
		FROM alpha_ranks ar
		JOIN trader_accounts ta ON ta.id = ar.account_id
		LEFT JOIN users u ON u.id = ta.user_id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		where, orderBy, argN, argN+1)

	rows, err := h.service.db.Query(sqlQ, queryArgs...)
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
		Layer3Multiplier float64 `json:"layer3Multiplier"`
		Layer3Status     string  `json:"layer3Status"`
		Layer3Reason     string  `json:"layer3Reason"`
		NetPnl       float64 `json:"netPnl"`
		TotalTrades  int     `json:"totalTrades"`
		ProfitFactor float64 `json:"profitFactor"`
		RiskLevel    string  `json:"riskLevel"`
		Strategy     string  `json:"strategy"`
		Currency     string  `json:"currency"`
	}
	var traders []TraderRow
	for rows.Next() {
		var t TraderRow
		rows.Scan(&t.ID, &t.Nickname, &t.Broker, &t.Platform,
			&t.AlphaScore, &t.Grade, &t.WinRate, &t.MaxDD,
			&t.ROI, &t.NetPnl, &t.TotalTrades, &t.ProfitFactor,
			&t.TraderName, &t.RiskLevel, &t.Strategy,
                    &t.Layer3Multiplier, &t.Layer3Status, &t.Layer3Reason, &t.Currency)
		traders = append(traders, t)
	}
	if traders == nil { traders = []TraderRow{} }

	totalPages := 1
	if total > 0 { totalPages = (total + limit - 1) / limit }
	c.JSON(200, gin.H{
		"ok":         true,
		"traders":    traders,
		"count":      len(traders),
		"total":      total,
		"page":       page,
		"totalPages": totalPages,
		"limit":      limit,
	})

}

// GET /api/public/trader/:id — public trader detail with per pair and flags
func (h *Handler) GetPublicTraderDetail(c *gin.Context) {
	accountID := c.Param("id")

	// Per pair
	pairRows, err := h.service.db.Query(`
                SELECT ar.symbol, ar.trade_count, ar.win_rate, ar.profit_factor,
                       COALESCE(ar.net_pnl,0), COALESCE(ar.risk_reward,0),
                       COALESCE(ar.risk_flags,'[]'::jsonb)
		FROM alpha_ranks ar
                WHERE account_id=$1::uuid AND ar.symbol != 'ALL' AND ar.trade_count >= 20
                ORDER BY ar.net_pnl DESC LIMIT 8`, accountID)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer pairRows.Close()
	type PairRow struct {
		Symbol       string        `json:"symbol"`
		Trades       int           `json:"trades"`
		WinRate      float64       `json:"winRate"`
		ProfitFactor float64       `json:"profitFactor"`
		NetPnl       float64       `json:"netPnl"`
		AvgRR        float64       `json:"avgRR"`
		Flags        []interface{} `json:"flags"`
	}
	var pairs []PairRow
	for pairRows.Next() {
		var p PairRow
		var flagsJSON []byte
		pairRows.Scan(&p.Symbol, &p.Trades, &p.WinRate, &p.ProfitFactor,
			&p.NetPnl, &p.AvgRR, &flagsJSON)
		if len(flagsJSON) > 0 { json.Unmarshal(flagsJSON, &p.Flags) }
		if p.Flags == nil { p.Flags = []interface{}{} }
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
                Layer3Multiplier float64
                Layer3Status     string
                Layer3Reason     string
	}
	h.service.db.QueryRow(`
		SELECT alpha_score, grade, COALESCE(tier,''), win_rate, profit_factor,
		       max_drawdown_pct, COALESCE(net_pnl,0), COALESCE(total_trades_all,0),
		       COALESCE(winning_trades,0), COALESCE(losing_trades,0),
		       COALESCE(avg_win,0), COALESCE(avg_loss,0),
		       COALESCE(risk_reward,0), COALESCE(expectancy,0),
		       COALESCE(roi,0), COALESCE(risk_level,'MEDIUM'),
		       COALESCE(survivability_score,0), COALESCE(scalability_score,0),
		       COALESCE(risk_flags,'[]'::jsonb), COALESCE(pillars,'[]'::jsonb),
                       COALESCE(layer3_multiplier,1.0), COALESCE(layer3_status,'NEUTRAL'), COALESCE(layer3_reason,'')
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
                &allStats.Layer3Multiplier, &allStats.Layer3Status, &allStats.Layer3Reason,
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
                "layer3Multiplier": allStats.Layer3Multiplier,
                "layer3Status":     allStats.Layer3Status,
                "layer3Reason":     allStats.Layer3Reason,
		"survivability": allStats.Survivability,
		"scalability":   allStats.Scalability,
		"flags":         flags,
		"pillars":       pillarsPublic,
		"pairs":         pairs,
	})
}
