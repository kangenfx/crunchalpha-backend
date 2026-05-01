package investor

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func genKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "ea-inv-" + hex.EncodeToString(b)
}

// GET /api/investor/settings
func (h *Handler) GetSettings(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	type Settings struct {
		CopySignalEnabled bool    `json:"copySignalEnabled"`
		SignalLotSize     float64 `json:"signalLotSize"`
		SignalMaxLot      float64 `json:"signalMaxLot"`
		SignalRiskPct     float64 `json:"signalRiskPct"`
		SignalLotMode     string  `json:"signalLotMode"`
		CopyTraderEnabled bool    `json:"copyTraderEnabled"`
		TraderLotSize     float64 `json:"traderLotSize"`
		TraderMaxLot      float64 `json:"traderMaxLot"`
		TraderRiskPct     float64 `json:"traderRiskPct"`
		TraderLotMode     string  `json:"traderLotMode"`
		MaxDailyLossPct   float64 `json:"maxDailyLossPct"`
		MaxOpenTrades     int     `json:"maxOpenTrades"`
		Mt5Account        string  `json:"mt5Account"`
		EaKey             string  `json:"eaKey"`
		RiskLevel         string  `json:"riskLevel"`
		InvestorEquity    float64 `json:"investorEquity"`
		UpdatedAt         string  `json:"updatedAt"`
	}

	var s Settings
	var updatedAt time.Time
	err := h.service.repo.DB.QueryRow(`
		SELECT copy_signal_enabled, signal_lot_size, signal_max_lot, signal_risk_percent,
		       COALESCE(signal_lot_mode,'FIXED'),
		       copy_trader_enabled, trader_lot_size, trader_max_lot, trader_risk_percent,
		       COALESCE(trader_lot_mode,'FIXED'),
		       max_daily_loss_pct, max_open_trades, mt5_account, ea_key,
			       COALESCE(risk_level,'balanced'), updated_at
		FROM investor_settings WHERE investor_id=$1::uuid`, uid).Scan(
		&s.CopySignalEnabled, &s.SignalLotSize, &s.SignalMaxLot, &s.SignalRiskPct, &s.SignalLotMode,
		&s.CopyTraderEnabled, &s.TraderLotSize, &s.TraderMaxLot, &s.TraderRiskPct, &s.TraderLotMode,
		&s.MaxDailyLossPct, &s.MaxOpenTrades, &s.Mt5Account, &s.EaKey, &s.RiskLevel, &updatedAt)

	if err != nil {
		// Return defaults if not found
		s = Settings{
			SignalLotSize: 0.01, SignalMaxLot: 0.10, SignalRiskPct: 1.0, SignalLotMode: "FIXED",
			TraderLotSize: 0.01, TraderMaxLot: 0.10, TraderRiskPct: 1.0, TraderLotMode: "FIXED",
			MaxDailyLossPct: 5.0, MaxOpenTrades: 10, RiskLevel: "balanced",
		}
	} else {
		s.UpdatedAt = updatedAt.Format("2006-01-02 15:04")
	}
	// Ambil investorEquity real-time dari investor_ea_keys — SUM per unique mt5_account
	h.service.repo.DB.QueryRow(
		`SELECT COALESCE(SUM(eq),0) FROM (
		  SELECT DISTINCT ON (mt5_account) equity as eq
		  FROM investor_ea_keys
		  WHERE investor_id=$1::uuid
		  ORDER BY mt5_account, last_equity_at DESC NULLS LAST
		) t`,
		uid).Scan(&s.InvestorEquity)
	c.JSON(200, gin.H{"ok": true, "settings": s})
}

// POST /api/investor/settings
func (h *Handler) SaveSettings(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	var req struct {
		CopySignalEnabled *bool    `json:"copySignalEnabled"`
		SignalLotSize     *float64 `json:"signalLotSize"`
		SignalMaxLot      *float64 `json:"signalMaxLot"`
		SignalRiskPct     *float64 `json:"signalRiskPct"`
		CopyTraderEnabled *bool    `json:"copyTraderEnabled"`
		TraderLotSize     *float64 `json:"traderLotSize"`
		TraderMaxLot      *float64 `json:"traderMaxLot"`
		TraderRiskPct     *float64 `json:"traderRiskPct"`
		MaxDailyLossPct   *float64 `json:"maxDailyLossPct"`
		MaxOpenTrades     *int     `json:"maxOpenTrades"`
		Mt5Account        *string  `json:"mt5Account"`
		SignalLotMode     *string  `json:"signalLotMode"`
		TraderLotMode     *string  `json:"traderLotMode"`
		RiskLevel         *string  `json:"riskLevel"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"ok": false, "error": "invalid request"}); return
	}
	if req.RiskLevel != nil {
		log.Printf("[Settings] riskLevel: %s", *req.RiskLevel)
	} else {
		log.Printf("[Settings] riskLevel: nil")
	}

	_, err := h.service.repo.DB.Exec(`
		INSERT INTO investor_settings
			(investor_id, copy_signal_enabled, signal_lot_size, signal_max_lot, signal_risk_percent, signal_lot_mode,
			 copy_trader_enabled, trader_lot_size, trader_max_lot, trader_risk_percent, trader_lot_mode,
			 max_daily_loss_pct, max_open_trades, mt5_account, risk_level, updated_at)
		VALUES ($1::uuid,
			COALESCE($2,false), COALESCE($3,0.01), COALESCE($4,0.10), COALESCE($5,1.0), COALESCE($13,'FIXED'),
			COALESCE($6,false), COALESCE($7,0.01), COALESCE($8,0.10), COALESCE($9,1.0), COALESCE($14,'FIXED'),
			COALESCE($10,5.0), COALESCE($11,10), COALESCE($12,''), COALESCE($15,'balanced'), now())
		ON CONFLICT (investor_id) DO UPDATE SET
			copy_signal_enabled = COALESCE($2, investor_settings.copy_signal_enabled),
			signal_lot_size     = COALESCE($3, investor_settings.signal_lot_size),
			signal_max_lot      = COALESCE($4, investor_settings.signal_max_lot),
			signal_risk_percent = COALESCE($5, investor_settings.signal_risk_percent),
			signal_lot_mode     = COALESCE($13, investor_settings.signal_lot_mode),
			copy_trader_enabled = COALESCE($6, investor_settings.copy_trader_enabled),
			trader_lot_size     = COALESCE($7, investor_settings.trader_lot_size),
			trader_max_lot      = COALESCE($8, investor_settings.trader_max_lot),
			trader_risk_percent = COALESCE($9, investor_settings.trader_risk_percent),
			trader_lot_mode     = COALESCE($14, investor_settings.trader_lot_mode),
			max_daily_loss_pct  = COALESCE($10, investor_settings.max_daily_loss_pct),
			max_open_trades     = COALESCE($11, investor_settings.max_open_trades),
			mt5_account         = COALESCE($12, investor_settings.mt5_account),
			risk_level          = COALESCE($15, investor_settings.risk_level),
			updated_at          = now()`,
		uid, req.CopySignalEnabled, req.SignalLotSize, req.SignalMaxLot, req.SignalRiskPct,
		req.CopyTraderEnabled, req.TraderLotSize, req.TraderMaxLot, req.TraderRiskPct,
		req.MaxDailyLossPct, req.MaxOpenTrades, req.Mt5Account, req.SignalLotMode, req.TraderLotMode, req.RiskLevel)

	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "save failed: "+err.Error()}); return }
	if req.MaxOpenTrades != nil {
		h.service.repo.DB.Exec(`UPDATE user_allocations SET max_positions=$2, updated_at=now() WHERE user_id=$1::uuid`, uid, *req.MaxOpenTrades)
		h.service.repo.DB.Exec(`UPDATE investor_ea_keys SET max_open_trades=$2 WHERE investor_id=$1::uuid`, uid, *req.MaxOpenTrades)
	}
	if req.RiskLevel != nil {
		h.service.repo.DB.Exec(`UPDATE investor_ea_keys SET risk_level=$2 WHERE investor_id=$1::uuid`, uid, *req.RiskLevel)
		// Auto-set max_daily_loss_pct berdasarkan risk level
		ddPct := 10.0
		switch *req.RiskLevel {
		case "conservative":
			ddPct = 5.0
		case "aggressive":
			ddPct = 20.0
		}
		h.service.repo.DB.Exec(`UPDATE investor_settings SET max_daily_loss_pct=$2 WHERE investor_id=$1::uuid`, uid, ddPct)
		h.service.repo.DB.Exec(`UPDATE investor_ea_keys SET max_daily_loss_pct=$2 WHERE investor_id=$1::uuid`, uid, ddPct)
	}
	c.JSON(200, gin.H{"ok": true, "message": "Settings saved"})
}

// POST /api/investor/settings/generate-key
func (h *Handler) GenerateEAKey(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	newKey := genKey()
	hashed := hashEAKey(newKey)

	_, err := h.service.repo.DB.Exec(`
		INSERT INTO investor_settings (investor_id, ea_key, updated_at)
		VALUES ($1::uuid, $2, now())
		ON CONFLICT (investor_id) DO UPDATE SET ea_key=$2, updated_at=now()`,
		uid, hashed)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "key generation failed"}); return }

	c.JSON(200, gin.H{"ok": true, "eaKey": newKey, "message": "Save this key — it will not be shown again!"})
}

// GET /api/ea/investor/settings — EA pulls config
func (h *Handler) EAGetSettings(c *gin.Context) {
	investorID := getEAInvestorID(c)
	if investorID == "" { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	var s struct {
		CopySignalEnabled bool    `json:"copySignalEnabled"`
		SignalLotSize     float64 `json:"signalLotSize"`
		SignalMaxLot      float64 `json:"signalMaxLot"`
		CopyTraderEnabled bool    `json:"copyTraderEnabled"`
		TraderLotSize     float64 `json:"traderLotSize"`
		TraderMaxLot      float64 `json:"traderMaxLot"`
		MaxDailyLossPct   float64 `json:"maxDailyLossPct"`
		MaxOpenTrades     int     `json:"maxOpenTrades"`
	}
	err := h.service.repo.DB.QueryRow(`
		SELECT copy_signal_enabled, signal_lot_size, signal_max_lot,
		       copy_trader_enabled, trader_lot_size, trader_max_lot,
		       max_daily_loss_pct, max_open_trades
		FROM investor_settings WHERE investor_id=$1::uuid`, investorID).Scan(
		&s.CopySignalEnabled, &s.SignalLotSize, &s.SignalMaxLot,
		&s.CopyTraderEnabled, &s.TraderLotSize, &s.TraderMaxLot,
		&s.MaxDailyLossPct, &s.MaxOpenTrades)

	if err != nil {
		// defaults
		s.SignalLotSize = 0.01; s.SignalMaxLot = 0.10
		s.TraderLotSize = 0.01; s.TraderMaxLot = 0.10
		s.MaxDailyLossPct = 5.0; s.MaxOpenTrades = 10
	}
	c.JSON(200, gin.H{"ok": true, "settings": s})
}

func hashEAKey(k string) string {
	import_sha := func() string {
		h := make([]byte, 32)
		return hex.EncodeToString(h)
	}
	_ = import_sha
	return k // store plaintext for now, hash later
}

// AutoAllocate — hitung dan save allocation berdasarkan AlphaScore trader
func (h *Handler) AutoAllocate(c *gin.Context) {
uid, ok := getUID(c)
if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

// Get semua active subscriptions investor
rows, err := h.service.repo.DB.Query(`
SELECT cs.provider_account_id::text, COALESCE(ar.alpha_score, 0) as alpha_score
FROM copy_subscriptions cs
LEFT JOIN alpha_ranks ar ON ar.account_id = cs.provider_account_id AND ar.symbol = 'ALL'
LEFT JOIN trader_accounts ta ON ta.id = cs.provider_account_id
WHERE ta.user_id != $1::uuid
AND cs.follower_account_id IN (
SELECT id FROM trader_accounts WHERE user_id = $1::uuid AND status = 'active'
)
AND cs.status = 'ACTIVE'
`, uid)
if err != nil { c.JSON(500, gin.H{"ok": false, "error": err.Error()}); return }
defer rows.Close()

type TraderAlloc struct {
TraderID   string
AlphaScore float64
}
var traders []TraderAlloc
totalScore := 0.0
for rows.Next() {
var t TraderAlloc
rows.Scan(&t.TraderID, &t.AlphaScore)
if t.AlphaScore <= 0 { t.AlphaScore = 1.0 } // min score 1
traders = append(traders, t)
totalScore += t.AlphaScore
}

if len(traders) == 0 {
c.JSON(200, gin.H{"ok": false, "error": "No active traders to allocate"})
return
}

// Hitung % per trader berdasarkan AlphaScore
// Round ke integer, pastikan total = 100
allocations := make([]map[string]interface{}, 0)
remaining := 100
for i, t := range traders {
var pct int
if i == len(traders)-1 {
pct = remaining // last trader gets remainder
} else {
pct = int(t.AlphaScore / totalScore * 100)
if pct < 1 { pct = 1 }
}
remaining -= pct
allocations = append(allocations, map[string]interface{}{
"trader_account_id": t.TraderID,
"pct":               pct,
"alpha_score":       t.AlphaScore,
})
}

// Save ke DB
for _, a := range allocations {
h.service.repo.DB.Exec(`
INSERT INTO user_allocations (user_id, trader_account_id, allocation_mode, allocation_value, status, updated_at)
VALUES ($1::uuid, $2::uuid, 'PERCENT', $3, 'ACTIVE', NOW())
ON CONFLICT (user_id, trader_account_id)
DO UPDATE SET allocation_value = $3, updated_at = NOW()
`, uid, a["trader_account_id"], a["pct"])
}

// Save allocation_mode = AUTO di settings
h.service.repo.DB.Exec(`
UPDATE investor_settings SET allocation_mode = 'AUTO', updated_at = NOW()
WHERE investor_id = $1::uuid
`, uid)

c.JSON(200, gin.H{"ok": true, "allocations": allocations, "message": "Auto allocation applied"})
}
