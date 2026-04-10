package admin

import (
	"database/sql"
	"net/http"
	"github.com/gin-gonic/gin"
)

type FeeHandler struct {
	DB *sql.DB
}

func NewFeeHandler(db *sql.DB) *FeeHandler {
	return &FeeHandler{DB: db}
}

type FeeOverride struct {
	ID                    string   `json:"id"`
	UserID                string   `json:"user_id"`
	UserEmail             string   `json:"user_email"`
	UserName              *string  `json:"user_name"`
	PerformanceFee        *float64 `json:"performance_fee"`
	SignalFeeMonthly      *float64 `json:"signal_fee_monthly"`
	PlatformFeeMonthly    *float64 `json:"platform_fee_monthly"`
	SubscriptionFee       *float64 `json:"subscription_fee_monthly"`
	RebateSharePct        *float64 `json:"rebate_share_pct"`
	AffiliateCommissionPct *float64 `json:"affiliate_commission_pct"`
	Note                  *string  `json:"note"`
	CreatedAt             string   `json:"created_at"`
}

// ListFeeOverrides - GET /api/admin/fee-overrides
func (h *FeeHandler) ListFeeOverrides(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT f.id, f.user_id, u.email, u.name,
		       f.performance_fee, f.signal_fee_monthly, f.platform_fee_monthly,
		       f.subscription_fee_monthly, f.rebate_share_pct, f.affiliate_commission_pct,
		       f.note, f.created_at
		FROM user_fee_overrides f
		JOIN users u ON u.id = f.user_id
		ORDER BY f.created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var overrides []FeeOverride
	for rows.Next() {
		var f FeeOverride
		err := rows.Scan(&f.ID, &f.UserID, &f.UserEmail, &f.UserName,
			&f.PerformanceFee, &f.SignalFeeMonthly, &f.PlatformFeeMonthly,
			&f.SubscriptionFee, &f.RebateSharePct, &f.AffiliateCommissionPct,
			&f.Note, &f.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		overrides = append(overrides, f)
	}
	if overrides == nil {
		overrides = []FeeOverride{}
	}
	c.JSON(http.StatusOK, gin.H{"data": overrides})
}

// UpsertFeeOverride - POST /api/admin/fee-overrides
func (h *FeeHandler) UpsertFeeOverride(c *gin.Context) {
	var req struct {
		UserEmail              string   `json:"user_email" binding:"required"`
		PerformanceFee         *float64 `json:"performance_fee"`
		SignalFeeMonthly       *float64 `json:"signal_fee_monthly"`
		PlatformFeeMonthly     *float64 `json:"platform_fee_monthly"`
		SubscriptionFee        *float64 `json:"subscription_fee_monthly"`
		RebateSharePct         *float64 `json:"rebate_share_pct"`
		AffiliateCommissionPct *float64 `json:"affiliate_commission_pct"`
		Note                   *string  `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var userID string
	err := h.DB.QueryRow(`SELECT id FROM users WHERE email = $1`, req.UserEmail).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found: " + req.UserEmail})
		return
	}

	var id string
	err = h.DB.QueryRow(`
		INSERT INTO user_fee_overrides
		    (user_id, performance_fee, signal_fee_monthly, platform_fee_monthly,
		     subscription_fee_monthly, rebate_share_pct, affiliate_commission_pct, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
		    performance_fee          = EXCLUDED.performance_fee,
		    signal_fee_monthly       = EXCLUDED.signal_fee_monthly,
		    platform_fee_monthly     = EXCLUDED.platform_fee_monthly,
		    subscription_fee_monthly = EXCLUDED.subscription_fee_monthly,
		    rebate_share_pct         = EXCLUDED.rebate_share_pct,
		    affiliate_commission_pct = EXCLUDED.affiliate_commission_pct,
		    note                     = EXCLUDED.note,
		    updated_at               = NOW()
		RETURNING id
	`, userID, req.PerformanceFee, req.SignalFeeMonthly, req.PlatformFeeMonthly,
		req.SubscriptionFee, req.RebateSharePct, req.AffiliateCommissionPct, req.Note).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "message": "Fee override saved"})
}

// DeleteFeeOverride - DELETE /api/admin/fee-overrides/:id
func (h *FeeHandler) DeleteFeeOverride(c *gin.Context) {
	id := c.Param("id")
	_, err := h.DB.Exec(`DELETE FROM user_fee_overrides WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Fee override deleted"})
}

// DefaultFees - GET /api/admin/default-fees
func (h *FeeHandler) GetDefaultFees(c *gin.Context) {
	keys := []string{
		"trader_performance_fee_pct",
		"analyst_performance_fee_pct",
		"trader_subscription_fee_usd",
		"analyst_subscription_fee_usd",
		"rebate_investor_investor_pct",
		"rebate_trader_ib_trader_pct",
		"affiliate_flat_pct",
	}
	result := map[string]float64{}
	for _, k := range keys {
		var v float64
		h.DB.QueryRow(`SELECT value FROM platform_fee_config WHERE key=$1`, k).Scan(&v)
		result[k] = v
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}
