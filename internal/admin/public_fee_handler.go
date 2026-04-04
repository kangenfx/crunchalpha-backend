package admin

import (
"database/sql"
"net/http"

"github.com/gin-gonic/gin"
)

type PublicFeeHandler struct {
db *sql.DB
}

func NewPublicFeeHandler(db *sql.DB) *PublicFeeHandler {
return &PublicFeeHandler{db: db}
}

func (h *PublicFeeHandler) GetPublicFees(c *gin.Context) {
keys := []string{
"trader_performance_fee_pct",
"trader_subscription_fee_usd",
"analyst_performance_fee_pct",
"analyst_subscription_fee_usd",
"max_traders_per_investor",
"max_analysts_per_investor",
"trader_split_trader_pct",
"analyst_split_analyst_pct",
}

fees := make(map[string]float64)
labels := make(map[string]string)

for _, key := range keys {
var value float64
var label string
err := h.db.QueryRow(`
SELECT value, label FROM platform_fee_config WHERE key = $1`, key,
).Scan(&value, &label)
if err != nil {
continue
}
fees[key] = value
labels[key] = label
}

c.JSON(http.StatusOK, gin.H{
"trader": gin.H{
"performance_fee_pct":  fees["trader_performance_fee_pct"],
"subscription_fee_usd": fees["trader_subscription_fee_usd"],
"trader_share_pct":     fees["trader_split_trader_pct"],
},
"analyst": gin.H{
"performance_fee_pct":  fees["analyst_performance_fee_pct"],
"subscription_fee_usd": fees["analyst_subscription_fee_usd"],
"analyst_share_pct":    fees["analyst_split_analyst_pct"],
},
"limits": gin.H{
"max_traders_per_investor":  fees["max_traders_per_investor"],
"max_analysts_per_investor": fees["max_analysts_per_investor"],
},
})
}
