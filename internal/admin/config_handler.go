package admin

import (
	"database/sql"
	"net/http"
	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	DB *sql.DB
}

func NewConfigHandler(db *sql.DB) *ConfigHandler {
	return &ConfigHandler{DB: db}
}

type FeeConfig struct {
	ID          string  `json:"id"`
	Key         string  `json:"key"`
	Value       float64 `json:"value"`
	Label       string  `json:"label"`
	Description *string `json:"description"`
	Category    string  `json:"category"`
	UpdatedAt   string  `json:"updated_at"`
}

type AffiliateT struct {
	ID                   string  `json:"id"`
	TierName             string  `json:"tier_name"`
	TierOrder            int     `json:"tier_order"`
	MinActiveReferrals   int     `json:"min_active_referrals"`
	MinActiveSubscribers int     `json:"min_active_subscribers"`
	MinRetentionPct      float64 `json:"min_retention_pct"`
	MaxChurnPct          float64 `json:"max_churn_pct"`
	MinAumProxyUsd       float64 `json:"min_aum_proxy_usd"`
	AffiliateSharePct    float64 `json:"affiliate_share_pct"`
	Color                string  `json:"color"`
}

// GET /api/admin/config/fees
func (h *ConfigHandler) ListFeeConfig(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, key, value, label, description, category, updated_at
		FROM platform_fee_config
		ORDER BY category, key
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()
	var configs []FeeConfig
	for rows.Next() {
		var f FeeConfig
		rows.Scan(&f.ID, &f.Key, &f.Value, &f.Label, &f.Description, &f.Category, &f.UpdatedAt)
		configs = append(configs, f)
	}
	if configs == nil { configs = []FeeConfig{} }
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

// PUT /api/admin/config/fees/:key
func (h *ConfigHandler) UpdateFeeConfig(c *gin.Context) {
	key := c.Param("key")
	userID := c.GetString("user_id")
	var req struct {
		Value       float64 `json:"value"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	_, err := h.DB.Exec(`
		UPDATE platform_fee_config
		SET value=$1, description=COALESCE($2,description),
		    updated_by=$3, updated_at=NOW()
		WHERE key=$4
	`, req.Value, req.Description, userID, key)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
}

// GET /api/admin/config/affiliate-tiers
func (h *ConfigHandler) ListAffiliateTiers(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, tier_name, tier_order,
		       min_active_referrals, min_active_subscribers,
		       min_retention_pct, max_churn_pct,
		       min_aum_proxy_usd, affiliate_share_pct, color
		FROM affiliate_tier_config
		ORDER BY tier_order
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()
	var tiers []AffiliateT
	for rows.Next() {
		var t AffiliateT
		rows.Scan(&t.ID, &t.TierName, &t.TierOrder,
			&t.MinActiveReferrals, &t.MinActiveSubscribers,
			&t.MinRetentionPct, &t.MaxChurnPct,
			&t.MinAumProxyUsd, &t.AffiliateSharePct, &t.Color)
		tiers = append(tiers, t)
	}
	if tiers == nil { tiers = []AffiliateT{} }
	c.JSON(http.StatusOK, gin.H{"data": tiers})
}

// PUT /api/admin/config/affiliate-tiers/:id
func (h *ConfigHandler) UpdateAffiliateTier(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		MinActiveReferrals   *int     `json:"min_active_referrals"`
		MinActiveSubscribers *int     `json:"min_active_subscribers"`
		MinRetentionPct      *float64 `json:"min_retention_pct"`
		MaxChurnPct          *float64 `json:"max_churn_pct"`
		MinAumProxyUsd       *float64 `json:"min_aum_proxy_usd"`
		AffiliateSharePct    *float64 `json:"affiliate_share_pct"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	_, err := h.DB.Exec(`
		UPDATE affiliate_tier_config SET
			min_active_referrals   = COALESCE($1, min_active_referrals),
			min_active_subscribers = COALESCE($2, min_active_subscribers),
			min_retention_pct      = COALESCE($3, min_retention_pct),
			max_churn_pct          = COALESCE($4, max_churn_pct),
			min_aum_proxy_usd      = COALESCE($5, min_aum_proxy_usd),
			affiliate_share_pct    = COALESCE($6, affiliate_share_pct),
			updated_at             = NOW()
		WHERE id = $7
	`, req.MinActiveReferrals, req.MinActiveSubscribers,
		req.MinRetentionPct, req.MaxChurnPct,
		req.MinAumProxyUsd, req.AffiliateSharePct, id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"message": "Tier updated"})
}

// GET /api/admin/config/docs
func (h *ConfigHandler) GetDocs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"business_model": gin.H{
			"title": "CrunchAlpha — Hybrid Business Model",
			"sections": []gin.H{
				{
					"title": "Platform Overview",
					"content": "CrunchAlpha adalah Hybrid Allocation Platform non-custodial yang menggabungkan Copy Trader (real account), Copy Analyst (signal), dan Investor multi akun, multi role. Dana investor tetap di broker masing-masing — platform tidak memegang dana.",
				},
				{
					"title": "Revenue Streams",
					"content": "1. Performance Fee Trader: 20% HWM dari profit copy trade. Split 70/20/10 (Trader/Platform/Affiliate).\n2. Performance Fee Analyst: 10% HWM dari profit auto-follow. Split 70/20/10.\n3. Subscription Analyst: ~$10/bulan per signal set. Split 70/20/10.\n4. Bonus Pool: 5-10% dari platform revenue share, dibagi ke top trader & analyst berdasarkan AlphaRank score.\n5. IB Rebates: Komisi dari broker rekanan per lot yang di-trade user via referral link kita.",
				},
				{
					"title": "Investor Tiers",
					"content": "Lite (< $1k equity akun investor):\n- Auto Execution: kena Performance Fee 10% HWM. Boleh pakai seluruh equity.\n- Manual Execution: TIDAK kena performance fee. Tidak ada cap exposure (Opsi C). Tetap bayar subscription $10/bulan/analis.\n- Analytics: Lite tetap dapat Regime, Win/Loss curve, DD curve, Risk profile, Survivability.\n\nPro (≥ $1k equity akun investor):\n- Auto & Manual: sama-sama kena performance fee 10% HWM untuk analis.\n- No cap, full intel + survivability + bonus pool.\n\nInstitutional (≥ $100k): Full features, priority support, custom fee negotiable. Phase 2.",
				},
				{
					"title": "Performance Fee — High Watermark (HWM)",
					"content": "HWM berarti performance fee hanya dikenakan pada NEW profit di atas peak equity sebelumnya. Contoh: equity peak $10,000. Turun ke $8,000 lalu naik ke $11,000 → fee hanya dari $1,000 (selisih dari peak $10,000 ke $11,000). Ini melindungi investor dari double-charging.\n\nIMPORTANT: Performance fee HANYA berlaku untuk Auto Execution. Manual Execution tidak kena performance fee sama sekali, baik Lite maupun Pro.",
				},
				{
					"title": "Affiliate System & Revenue Split",
					"content": "Affiliate share diambil dari jatah platform, bukan tambahan. Contoh Trader pakai IB, affiliate Bronze (3%): Trader 60% + Platform 37% + Affiliate 3% = 100%.\n\nKondisi split:\n1. Trader pakai IB: Trader 60% fixed, Platform max 40% (affiliate ambil dari sini)\n2. Trader tidak pakai IB: Trader 35% fixed, Platform 65% (tidak ada potongan affiliate)\n3. Investor/Signal pakai IB: Analyst 35% + Investor 15% fixed, Platform max 80% (affiliate ambil dari sini)\n\nTier affiliate berdasarkan kualitas referral: active referrals, active subscribers, retention rate, churn rate, estimated AUM proxy. Tier: Bronze → Silver → Gold → Platinum. Anti-abuse: fake accounts, circular referrals, one-day churn, broker rebate arbitrage diblokir otomatis.",
				},
				{
					"title": "Non-Custodial Architecture",
					"content": "Investor membuka akun langsung di broker rekanan via IB link CrunchAlpha. Dana tidak pernah masuk ke wallet CrunchAlpha. Copy trading berjalan via EA (Expert Advisor) di MetaTrader 4/5. Platform hanya mengelola signal routing, fee calculation, dan analytics.",
				},
				{
					"title": "Manual Execution Disclaimer",
					"content": "Manual Execution Disclaimer (wajib ditampilkan ke user):\n\nDengan memilih Manual Execution, kamu memilih untuk mengeksekusi sinyal secara mandiri. Platform tidak akan mengenakan Performance Fee untuk posisi manual.\n\nRisiko sepenuhnya ditanggung oleh investor. Platform tidak bertanggung jawab atas kerugian yang timbul dari keputusan eksekusi manual. Tidak ada cap exposure untuk manual execution — investor bertanggung jawab penuh atas position sizing.",
				},
				{
					"title": "AlphaRank Scoring",
					"content": "7 Pillar scoring system: P1 Profitability, P2 Consistency, P3 Risk Management, P4 Recovery, P5 Trading Edge, P6 Discipline, P7 Track Record. Score 0-100, Grade A-F. Dipakai untuk: Marketplace ranking, Bonus Pool distribution, Investor suitability assessment.",
				},
			},
		},
	})
}
