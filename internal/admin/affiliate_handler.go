package admin

import (
"database/sql"
"net/http"

"github.com/gin-gonic/gin"
)

type AffiliateHandler struct {
DB *sql.DB
}

func NewAffiliateHandler(db *sql.DB) *AffiliateHandler {
return &AffiliateHandler{DB: db}
}

func (h *AffiliateHandler) ListAffiliates(c *gin.Context) {
rows, err := h.DB.Query(`
SELECT
a.id::text, a.code, a.tier,
a.total_referrals, a.active_referrals, a.eum_usd,
COALESCE(a.custom_commission_pct, pf.value) AS commission_pct,
a.custom_commission_pct IS NOT NULL AS is_custom,
a.created_at::text,
u.email, COALESCE(u.full_name,'') AS full_name, COALESCE(u.primary_role,'') AS role,
COALESCE(paid.total,0) AS total_paid,
COALESCE(pending.total,0) AS total_pending
FROM affiliates a
JOIN users u ON u.id = a.user_id
CROSS JOIN (SELECT value FROM platform_fee_config WHERE key='affiliate_flat_pct') pf
LEFT JOIN (
SELECT affiliate_id, SUM(amount_usd) AS total
FROM affiliate_payouts WHERE status='PAID'
GROUP BY affiliate_id
) paid ON paid.affiliate_id = a.id
LEFT JOIN (
SELECT affiliate_id, SUM(amount_usd) AS total
FROM affiliate_payouts WHERE status='PENDING'
GROUP BY affiliate_id
) pending ON pending.affiliate_id = a.id
ORDER BY a.created_at DESC
`)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
return
}
defer rows.Close()

type AffRow struct {
ID              string  `json:"id"`
Code            string  `json:"code"`
Tier            string  `json:"tier"`
TotalReferrals  int     `json:"total_referrals"`
ActiveReferrals int     `json:"active_referrals"`
EumUSD          float64 `json:"eum_usd"`
CommissionPct   float64 `json:"commission_pct"`
IsCustom        bool    `json:"is_custom"`
CreatedAt       string  `json:"created_at"`
Email           string  `json:"email"`
FullName        string  `json:"full_name"`
Role            string  `json:"role"`
TotalPaid       float64 `json:"total_paid"`
TotalPending    float64 `json:"total_pending"`
}

var list []AffRow
for rows.Next() {
var r AffRow
rows.Scan(&r.ID, &r.Code, &r.Tier,
&r.TotalReferrals, &r.ActiveReferrals, &r.EumUSD,
&r.CommissionPct, &r.IsCustom, &r.CreatedAt,
&r.Email, &r.FullName, &r.Role,
&r.TotalPaid, &r.TotalPending)
list = append(list, r)
}
if list == nil {
list = []AffRow{}
}

var totalCount, activeCount int
var totalEUM, totalPaid, totalPending float64
h.DB.QueryRow(`SELECT COUNT(*), COALESCE(SUM(active_referrals),0), COALESCE(SUM(eum_usd),0) FROM affiliates`).
Scan(&totalCount, &activeCount, &totalEUM)
h.DB.QueryRow(`SELECT COALESCE(SUM(amount_usd),0) FROM affiliate_payouts WHERE status='PAID'`).Scan(&totalPaid)
h.DB.QueryRow(`SELECT COALESCE(SUM(amount_usd),0) FROM affiliate_payouts WHERE status='PENDING'`).Scan(&totalPending)

var mode float64
var flatPct float64
h.DB.QueryRow(`SELECT value FROM platform_fee_config WHERE key='affiliate_mode'`).Scan(&mode)
h.DB.QueryRow(`SELECT value FROM platform_fee_config WHERE key='affiliate_flat_pct'`).Scan(&flatPct)

c.JSON(http.StatusOK, gin.H{
"ok":   true,
"data": list,
"summary": gin.H{
"total_affiliates":  totalCount,
"total_referrals":   activeCount,
"total_eum_usd":     totalEUM,
"total_paid_usd":    totalPaid,
"total_pending_usd": totalPending,
},
"config": gin.H{
"mode":     mode,
"flat_pct": flatPct,
},
})
}

func (h *AffiliateHandler) SetCustomCommission(c *gin.Context) {
id := c.Param("id")
var req struct {
CustomPct *float64 `json:"custom_pct"`
}
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
return
}
_, err := h.DB.Exec(`UPDATE affiliates SET custom_commission_pct=$1 WHERE id=$2::uuid`, req.CustomPct, id)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
return
}
c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Commission updated"})
}

func (h *AffiliateHandler) AddPayout(c *gin.Context) {
id := c.Param("id")
var req struct {
AmountUSD float64 `json:"amount_usd"`
Source    string  `json:"source"`
Period    string  `json:"period"`
Status    string  `json:"status"`
}
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
return
}
if req.Status == "" {
req.Status = "PENDING"
}
_, err := h.DB.Exec(`
INSERT INTO affiliate_payouts (id, affiliate_id, amount_usd, source, status, period)
VALUES (gen_random_uuid(), $1::uuid, $2, $3, $4, $5)
`, id, req.AmountUSD, req.Source, req.Status, req.Period)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
return
}
c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Payout recorded"})
}

func (h *AffiliateHandler) MarkPayoutPaid(c *gin.Context) {
payoutID := c.Param("payout_id")
_, err := h.DB.Exec(`UPDATE affiliate_payouts SET status='PAID' WHERE id=$1::uuid`, payoutID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
return
}
c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Payout marked as paid"})
}

func (h *AffiliateHandler) UpdateAffiliateConfig(c *gin.Context) {
var req struct {
Mode    *float64 `json:"mode"`
FlatPct *float64 `json:"flat_pct"`
}
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
return
}
if req.Mode != nil {
h.DB.Exec(`UPDATE platform_fee_config SET value=$1, updated_at=NOW() WHERE key='affiliate_mode'`, *req.Mode)
}
if req.FlatPct != nil {
h.DB.Exec(`UPDATE platform_fee_config SET value=$1, updated_at=NOW() WHERE key='affiliate_flat_pct'`, *req.FlatPct)
}
c.JSON(http.StatusOK, gin.H{"ok": true, "message": "Affiliate config updated"})
}
