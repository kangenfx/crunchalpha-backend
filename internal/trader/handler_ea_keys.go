package trader

import (
"fmt"
"net/http"
"time"

"crunchalpha-v3/internal/apikey"
"github.com/gin-gonic/gin"
)

// GET /api/trader/ea-keys
func (h *Handler) GetTraderEAKeys(c *gin.Context) {
userID := fmt.Sprintf("%v", c.GetString("user_id"))

rows, err := h.DB.Query(`
SELECT id, name, key_prefix, created_at, last_used_at
FROM api_keys
WHERE user_id = $1 AND active = true AND name LIKE 'EA Key%'
ORDER BY created_at DESC`, userID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
return
}
defer rows.Close()

type KeyInfo struct {
ID        string     `json:"id"`
Name      string     `json:"name"`
KeyPrefix string     `json:"key_prefix"`
Platform  string     `json:"platform"`
CreatedAt time.Time  `json:"created_at"`
LastUsed  *time.Time `json:"last_used"`
}

keys := []KeyInfo{}
for rows.Next() {
var k KeyInfo
if err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.CreatedAt, &k.LastUsed); err != nil {
continue
}
k.Platform = "MT5"
keys = append(keys, k)
}
c.JSON(http.StatusOK, gin.H{"ok": true, "keys": keys})
}

// POST /api/trader/ea-keys
func (h *Handler) GenerateTraderEAKey(c *gin.Context) {
userID := fmt.Sprintf("%v", c.GetString("user_id"))

var body struct {
AccountID   string `json:"account_id"`
Platform    string `json:"platform"`
Description string `json:"description"`
}
c.ShouldBindJSON(&body)

// Max 3 keys
var count int
h.DB.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE user_id=$1 AND active=true AND name LIKE 'EA Key%'`, userID).Scan(&count)
if count >= 10 {
c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "Maximum 3 EA keys allowed"})
return
}

plainKey, err := apikey.GenerateAPIKey()
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to generate key"})
return
}
keyHash := apikey.HashAPIKey(plainKey)
keyPrefix := apikey.GetKeyPrefix(plainKey)

platform := body.Platform
if platform == "" {
platform = "MT5"
}
name := fmt.Sprintf("EA Key — %s %s", platform, body.AccountID)

// Insert ke api_keys
var keyID string
err = h.DB.QueryRow(`
INSERT INTO api_keys (user_id, key_hash, key_prefix, name, active)
VALUES ($1, $2, $3, $4, true)
RETURNING id`,
userID, keyHash, keyPrefix, name,
).Scan(&keyID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to save key"})
return
}

// Link ke account via api_key_accounts
if body.AccountID != "" {
h.DB.Exec(`INSERT INTO api_key_accounts (api_key_id, account_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING`, keyID, body.AccountID)
}

c.JSON(http.StatusOK, gin.H{
"ok":     true,
"key":    plainKey,
"key_id": keyID,
})
}

// DELETE /api/trader/ea-keys/:id
func (h *Handler) DeleteTraderEAKey(c *gin.Context) {
userID := fmt.Sprintf("%v", c.GetString("user_id"))
keyID := c.Param("id")

res, err := h.DB.Exec(`
UPDATE api_keys SET active=false, revoked_at=now()
WHERE id=$1 AND user_id=$2`, keyID, userID)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "db error"})
return
}
rows, _ := res.RowsAffected()
if rows == 0 {
c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "key not found"})
return
}
c.JSON(http.StatusOK, gin.H{"ok": true})
}
