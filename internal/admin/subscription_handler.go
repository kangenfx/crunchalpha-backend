package admin

import (
"database/sql"
"fmt"
"time"

"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
DB *sql.DB
}

// ── GET /api/admin/subscriptions ─────────────────────────────────────────────

func NewSubscriptionHandler(db *sql.DB) *SubscriptionHandler {
	return &SubscriptionHandler{DB: db}
}

func (h *SubscriptionHandler) ListSubscriptions(c *gin.Context) {
rows, err := h.DB.Query(`
SELECT id, COALESCE(name,''), email, primary_role,
       COALESCE(subscription_tier,'none'),
       COALESCE(subscription_status,'none'),
       trial_started_at, trial_ends_at,
       subscription_started_at, next_billing_date,
       created_at
FROM users
WHERE primary_role IN ('investor','both')
ORDER BY created_at DESC`)
if err != nil {
c.JSON(500, gin.H{"ok": false, "error": "db error"})
return
}
defer rows.Close()

type SubRow struct {
ID                  string  `json:"id"`
Name                string  `json:"name"`
Email               string  `json:"email"`
Role                string  `json:"role"`
Tier                string  `json:"tier"`
Status              string  `json:"status"`
TrialStartedAt      *string `json:"trial_started_at"`
TrialEndsAt         *string `json:"trial_ends_at"`
SubStartedAt        *string `json:"sub_started_at"`
NextBillingDate     *string `json:"next_billing_date"`
CreatedAt           string  `json:"created_at"`
}

list := make([]SubRow, 0)
for rows.Next() {
var r SubRow
var trialStart, trialEnd, subStart, nextBill sql.NullTime
var createdAt time.Time
rows.Scan(&r.ID, &r.Name, &r.Email, &r.Role,
&r.Tier, &r.Status,
&trialStart, &trialEnd, &subStart, &nextBill,
&createdAt)
r.CreatedAt = createdAt.Format("2006-01-02")
if trialStart.Valid { s := trialStart.Time.Format("2006-01-02"); r.TrialStartedAt = &s }
if trialEnd.Valid   { s := trialEnd.Time.Format("2006-01-02");   r.TrialEndsAt = &s }
if subStart.Valid   { s := subStart.Time.Format("2006-01-02");   r.SubStartedAt = &s }
if nextBill.Valid   { s := nextBill.Time.Format("2006-01-02");   r.NextBillingDate = &s }
list = append(list, r)
}
c.JSON(200, gin.H{"ok": true, "subscriptions": list, "count": len(list)})
}

// ── PUT /api/admin/subscriptions/:userId ─────────────────────────────────────
func (h *SubscriptionHandler) UpdateSubscription(c *gin.Context) {
userId := c.Param("userId")
var req struct {
Tier            string `json:"tier"`
Status          string `json:"status"`
FreeSubscription bool  `json:"free_subscription"`
NextBillingDate string `json:"next_billing_date"`
TrialDays       int    `json:"trial_days"`
}
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(400, gin.H{"ok": false, "error": "invalid json"})
return
}

validTiers := map[string]bool{"none": true, "trial": true, "basic": true, "premium": true}
if req.Tier != "" && !validTiers[req.Tier] {
c.JSON(400, gin.H{"ok": false, "error": "invalid tier: none/trial/basic/premium"})
return
}

// Build update
now := time.Now()
var dbErr error

if req.Tier == "trial" {
days := req.TrialDays
if days <= 0 { days = 14 }
trialEnd := now.AddDate(0, 0, days)
_, dbErr = h.DB.Exec(`
UPDATE users SET
subscription_tier='trial',
subscription_status='active',
trial_started_at=$1,
trial_ends_at=$2,
subscription_started_at=$1,
updated_at=now()
WHERE id=$3`, now, trialEnd, userId)
} else if req.Tier == "basic" || req.Tier == "premium" {
status := "active"
if req.FreeSubscription { status = "free" }
var nextBill interface{}
if req.NextBillingDate != "" {
if t, err := time.Parse("2006-01-02", req.NextBillingDate); err == nil {
nextBill = t
}
} else {
nextBill = now.AddDate(0, 1, 0) // default 1 bulan
}
_, dbErr = h.DB.Exec(`
UPDATE users SET
subscription_tier=$1,
subscription_status=$2,
subscription_started_at=$3,
next_billing_date=$4,
updated_at=now()
WHERE id=$5`, req.Tier, status, now, nextBill, userId)
} else if req.Tier == "none" {
_, dbErr = h.DB.Exec(`
UPDATE users SET
subscription_tier='none',
subscription_status='none',
subscription_started_at=NULL,
next_billing_date=NULL,
trial_started_at=NULL,
trial_ends_at=NULL,
updated_at=now()
WHERE id=$1`, userId)
} else if req.Status != "" {
// Only update status
_, dbErr = h.DB.Exec(`UPDATE users SET subscription_status=$1, updated_at=now() WHERE id=$2`,
req.Status, userId)
}

if dbErr != nil {
c.JSON(500, gin.H{"ok": false, "error": fmt.Sprintf("db error: %v", dbErr)})
return
}
c.JSON(200, gin.H{"ok": true, "user_id": userId, "tier": req.Tier, "status": req.Status})
}

// ── GET /api/admin/subscription-stats ────────────────────────────────────────
func (h *SubscriptionHandler) GetStats(c *gin.Context) {
type Stats struct {
Total   int `json:"total"`
None    int `json:"none"`
Trial   int `json:"trial"`
Basic   int `json:"basic"`
Premium int `json:"premium"`
Free    int `json:"free"`
}
var s Stats
h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role IN ('investor','both')`).Scan(&s.Total)
h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role IN ('investor','both') AND COALESCE(subscription_tier,'none')='none'`).Scan(&s.None)
h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role IN ('investor','both') AND subscription_tier='trial'`).Scan(&s.Trial)
h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role IN ('investor','both') AND subscription_tier='basic'`).Scan(&s.Basic)
h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role IN ('investor','both') AND subscription_tier='premium'`).Scan(&s.Premium)
h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role IN ('investor','both') AND subscription_status='free'`).Scan(&s.Free)
c.JSON(200, gin.H{"ok": true, "stats": s})
}
