package admin

import (
	"database/sql"
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
)

type FeeCalculator struct {
	DB *sql.DB
}

func NewFeeCalculator(db *sql.DB) *FeeCalculator {
	return &FeeCalculator{DB: db}
}

type FeeSimResult struct {
	AccountID       string  `json:"account_id"`
	AccountNumber   string  `json:"account_number"`
	Broker          string  `json:"broker"`
	UserEmail       string  `json:"user_email"`
	UsesPlatformIB  bool    `json:"uses_platform_ib"`
	IBBrokerName    string  `json:"ib_broker_name"`

	// Trading stats
	TotalLots       float64 `json:"total_lots"`
	TotalProfit     float64 `json:"total_profit"`
	PeakEquity      float64 `json:"peak_equity"`
	CurrentEquity   float64 `json:"current_equity"`
	NewProfit       float64 `json:"new_profit_above_hwm"`

	// Config used
	PerfFeePct      float64 `json:"perf_fee_pct"`
	TraderSharePct  float64 `json:"trader_share_pct"`
	PlatformMaxPct  float64 `json:"platform_max_pct"`
	BrokerPaysPerLot float64 `json:"broker_pays_per_lot"`

	// Calculated fees
	GrossRebate     float64 `json:"gross_rebate"`
	PerfFeeTotal    float64 `json:"perf_fee_total"`

	// Split breakdown
	TraderGets      float64 `json:"trader_gets"`
	PlatformGets    float64 `json:"platform_gets"`
	AffiliateGets   float64 `json:"affiliate_gets"`
	AffiliateSharePct float64 `json:"affiliate_share_pct"`
	AffiliateTier   string  `json:"affiliate_tier"`

	// Period
	PeriodFrom      string  `json:"period_from"`
	PeriodTo        string  `json:"period_to"`
}

// GET /api/admin/fee-simulation
func (h *FeeCalculator) SimulateFees(c *gin.Context) {
	accountID := c.Query("account_id")
	periodFrom := c.Query("from") // YYYY-MM-DD
	periodTo := c.Query("to")

	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id required"})
		return
	}

	// Default period = last 30 days
	if periodFrom == "" {
		periodFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if periodTo == "" {
		periodTo = time.Now().Format("2006-01-02")
	}

	// 1. Get account info
	var res FeeSimResult
	var ibBrokerID *string
	err := h.DB.QueryRow(`
		SELECT ta.id, ta.account_number, ta.broker, ta.equity,
		       ta.uses_platform_ib, COALESCE(ib.name,'') as ib_broker_name,
		       COALESCE(ib.broker_pays_per_lot, 0) as broker_pays,
		       ta.ib_broker_id::text, u.email
		FROM trader_accounts ta
		LEFT JOIN ib_brokers ib ON ib.id = ta.ib_broker_id
		LEFT JOIN users u ON u.id = ta.user_id
		WHERE ta.id = $1
	`, accountID).Scan(&res.AccountID, &res.AccountNumber, &res.Broker,
		&res.CurrentEquity, &res.UsesPlatformIB, &res.IBBrokerName,
		&res.BrokerPaysPerLot, &ibBrokerID, &res.UserEmail)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found: " + err.Error()})
		return
	}

	// 2. Get trading stats for period
	h.DB.QueryRow(`
		SELECT COALESCE(SUM(lots), 0), COALESCE(SUM(profit), 0)
		FROM trades
		WHERE account_id = $1
		AND close_time BETWEEN $2 AND $3
		AND close_time IS NOT NULL
	`, accountID, periodFrom, periodTo+" 23:59:59").Scan(&res.TotalLots, &res.TotalProfit)

	// 3. Get peak equity (HWM) from equity snapshots
	h.DB.QueryRow(`
		SELECT COALESCE(MAX(equity), 0)
		FROM equity_snapshots
		WHERE account_id = $1
		AND created_at < $2
	`, accountID, periodFrom).Scan(&res.PeakEquity)

	// New profit above HWM
	if res.CurrentEquity > res.PeakEquity {
		res.NewProfit = res.CurrentEquity - res.PeakEquity
	}

	// 4. Get fee config from DB
	configs := make(map[string]float64)
	rows, _ := h.DB.Query(`SELECT key, value FROM platform_fee_config`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var k string
			var v float64
			rows.Scan(&k, &v)
			configs[k] = v
		}
	}

	// 5. Determine split based on IB status
	// Performance fee split = always 70/20/10 regardless IB status
	// IB status only affects REBATE split
	res.PerfFeePct = configs["trader_performance_fee_pct"]
	perfTraderPct := configs["trader_split_trader_pct"]     // 70%
	perfPlatformPct := configs["trader_split_platform_pct"] // 20%
	if perfTraderPct == 0 { perfTraderPct = 70 }
	if perfPlatformPct == 0 { perfPlatformPct = 20 }

	// Rebate split based on IB status
	var rebateTraderPct, rebatePlatformPct float64
	if res.UsesPlatformIB {
		rebateTraderPct = configs["rebate_trader_ib_trader_pct"]
		rebatePlatformPct = configs["rebate_trader_ib_platform_pct"]
	} else {
		rebateTraderPct = 0
		rebatePlatformPct = 0
	}
	res.TraderSharePct = perfTraderPct
	res.PlatformMaxPct = perfPlatformPct

	// 6. Get affiliate tier for this user
	var affiliateSharePct float64
	var affiliateTier string
	err2 := h.DB.QueryRow(`
		SELECT atc.affiliate_share_pct, atc.tier_name
		FROM affiliates af
		JOIN affiliate_tier_config atc ON atc.tier_name = af.tier
		JOIN trader_accounts ta ON ta.user_id = af.user_id
		WHERE ta.id = $1
	`, accountID).Scan(&affiliateSharePct, &affiliateTier)
	if err2 != nil {
		affiliateSharePct = 0
		affiliateTier = "None"
	}
	res.AffiliateSharePct = affiliateSharePct
	res.AffiliateTier = affiliateTier

	// 7. Calculate fees
	// Gross rebate from broker
	res.GrossRebate = res.TotalLots * res.BrokerPaysPerLot

	// Performance fee (HWM basis — only on new profit)
	res.PerfFeeTotal = res.NewProfit * (res.PerfFeePct / 100)

	// Split performance fee: 70/20/10
	perfTraderGets := res.PerfFeeTotal * (perfTraderPct / 100)
	perfAffiliateGets := res.PerfFeeTotal * (affiliateSharePct / 100)
	perfPlatformGets := res.PerfFeeTotal * ((perfPlatformPct - affiliateSharePct) / 100)
	if perfPlatformGets < 0 { perfPlatformGets = 0 }

	// Split rebate: based on IB status
	rebateAffiliateGets := res.GrossRebate * (affiliateSharePct / 100)
	rebateEffPlatformPct := rebatePlatformPct - affiliateSharePct
	if rebateEffPlatformPct < 0 { rebateEffPlatformPct = 0 }
	rebateTraderGets := res.GrossRebate * (rebateTraderPct / 100)
	rebatePlatformGets := res.GrossRebate * (rebateEffPlatformPct / 100)

	// Total
	res.TraderGets = perfTraderGets + rebateTraderGets
	res.AffiliateGets = perfAffiliateGets + rebateAffiliateGets
	res.PlatformGets = perfPlatformGets + rebatePlatformGets

	res.PeriodFrom = periodFrom
	res.PeriodTo = periodTo

	c.JSON(http.StatusOK, res)
}

// GET /api/admin/fee-simulation/all — semua accounts
func (h *FeeCalculator) SimulateAllAccounts(c *gin.Context) {
	periodFrom := c.Query("from")
	periodTo := c.Query("to")
	if periodFrom == "" { periodFrom = time.Now().AddDate(0, -1, 0).Format("2006-01-02") }
	if periodTo == "" { periodTo = time.Now().Format("2006-01-02") }

	rows, err := h.DB.Query(`
		SELECT id FROM trader_accounts WHERE status='active' ORDER BY created_at
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []FeeSimResult
	var ids []string
	for rows.Next() {
		var id string
		rows.Scan(&id)
		ids = append(ids, id)
	}

	// Reuse SimulateFees logic per account
	for _, id := range ids {
		c.Request.URL.RawQuery = "account_id=" + id + "&from=" + periodFrom + "&to=" + periodTo
		// inline calculate
		var res FeeSimResult
		var ibBrokerID *string
		err := h.DB.QueryRow(`
			SELECT ta.id, ta.account_number, ta.broker, ta.equity,
			       ta.uses_platform_ib, COALESCE(ib.name,'') as ib_broker_name,
			       COALESCE(ib.broker_pays_per_lot, 0) as broker_pays,
			       ta.ib_broker_id::text, u.email
			FROM trader_accounts ta
			LEFT JOIN ib_brokers ib ON ib.id = ta.ib_broker_id
			LEFT JOIN users u ON u.id = ta.user_id
			WHERE ta.id = $1
		`, id).Scan(&res.AccountID, &res.AccountNumber, &res.Broker,
			&res.CurrentEquity, &res.UsesPlatformIB, &res.IBBrokerName,
			&res.BrokerPaysPerLot, &ibBrokerID, &res.UserEmail)
		if err != nil { continue }

		h.DB.QueryRow(`
			SELECT COALESCE(SUM(lots),0), COALESCE(SUM(profit),0)
			FROM trades WHERE account_id=$1
			AND close_time BETWEEN $2 AND $3 AND close_time IS NOT NULL
		`, id, periodFrom, periodTo+" 23:59:59").Scan(&res.TotalLots, &res.TotalProfit)

		h.DB.QueryRow(`
			SELECT COALESCE(MAX(equity),0) FROM equity_snapshots
			WHERE account_id=$1 AND created_at < $2
		`, id, periodFrom).Scan(&res.PeakEquity)

		if res.CurrentEquity > res.PeakEquity {
			res.NewProfit = res.CurrentEquity - res.PeakEquity
		}

		configs := make(map[string]float64)
		cfgRows, _ := h.DB.Query(`SELECT key, value FROM platform_fee_config`)
		if cfgRows != nil {
			for cfgRows.Next() {
				var k string; var v float64
				cfgRows.Scan(&k, &v)
				configs[k] = v
			}
			cfgRows.Close()
		}

		res.PerfFeePct = configs["trader_performance_fee_pct"]
		perfTraderPct2 := configs["trader_split_trader_pct"]
		perfPlatformPct2 := configs["trader_split_platform_pct"]
		if perfTraderPct2 == 0 { perfTraderPct2 = 70 }
		if perfPlatformPct2 == 0 { perfPlatformPct2 = 20 }
		var rebateTraderPct2, rebatePlatformPct2 float64
		if res.UsesPlatformIB {
			rebateTraderPct2 = configs["rebate_trader_ib_trader_pct"]
			rebatePlatformPct2 = configs["rebate_trader_ib_platform_pct"]
		}
		res.TraderSharePct = perfTraderPct2
		res.PlatformMaxPct = perfPlatformPct2

		var affSharePct float64
		var affTier string
		h.DB.QueryRow(`
			SELECT atc.affiliate_share_pct, atc.tier_name
			FROM affiliates af
			JOIN affiliate_tier_config atc ON atc.tier_name = af.tier
			JOIN trader_accounts ta ON ta.user_id = af.user_id
			WHERE ta.id = $1
		`, id).Scan(&affSharePct, &affTier)
		if affTier == "" { affTier = "None" }
		res.AffiliateSharePct = affSharePct
		res.AffiliateTier = affTier

		res.GrossRebate = res.TotalLots * res.BrokerPaysPerLot
		res.PerfFeeTotal = res.NewProfit * (res.PerfFeePct / 100)
		pTG := res.PerfFeeTotal * (perfTraderPct2 / 100)
		pAG := res.PerfFeeTotal * (affSharePct / 100)
		pPG := res.PerfFeeTotal * ((perfPlatformPct2 - affSharePct) / 100)
		if pPG < 0 { pPG = 0 }
		rAG := res.GrossRebate * (affSharePct / 100)
		rEffP := rebatePlatformPct2 - affSharePct
		if rEffP < 0 { rEffP = 0 }
		rTG := res.GrossRebate * (rebateTraderPct2 / 100)
		rPG := res.GrossRebate * (rEffP / 100)
		res.TraderGets = pTG + rTG
		res.AffiliateGets = pAG + rAG
		res.PlatformGets = pPG + rPG
		res.PeriodFrom = periodFrom
		res.PeriodTo = periodTo

		results = append(results, res)
	}
	if results == nil { results = []FeeSimResult{} }
	c.JSON(http.StatusOK, gin.H{"data": results, "period_from": periodFrom, "period_to": periodTo})
}

// PUT /api/admin/trading-accounts/:id/ib-status — toggle IB status untuk testing
func (h *FeeCalculator) UpdateIBStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		UsesPlatformIB bool    `json:"uses_platform_ib"`
		IBBrokerID     *string `json:"ib_broker_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.DB.Exec(`
		UPDATE trader_accounts SET
			uses_platform_ib = $1,
			ib_broker_id = $2::uuid,
			updated_at = NOW()
		WHERE id = $3
	`, req.UsesPlatformIB, req.IBBrokerID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "IB status updated"})
}
