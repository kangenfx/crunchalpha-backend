package investor

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// EAKeyInfo — hasil lookup EA key dari DB
type EAKeyInfo struct {
	InvestorID  string
	KeyID       string
	MT5Account  string
	Platform    string
	Description string
}

// hashEAKeyV2 — SHA256 hash untuk EA key
func hashEAKeyV2(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// lookupEAKey — cari investor dari EA key header
func lookupEAKey(db *sql.DB, c *gin.Context) (*EAKeyInfo, error) {
	// Support both X-EA-Key (new) and X-Investor-ID (legacy)
	rawKey := c.GetHeader("X-EA-Key")
	legacyID := c.GetHeader("X-Investor-ID")

	if rawKey != "" {
		// New: lookup by EA key hash
		keyHash := hashEAKeyV2(rawKey)
		var info EAKeyInfo
		err := db.QueryRow(
			`SELECT id, investor_id::text, COALESCE(mt5_account,''), COALESCE(platform,'MT5'), COALESCE(description,'')
			 FROM investor_ea_keys
			 WHERE key_hash = $1`,
			keyHash).Scan(&info.KeyID, &info.InvestorID, &info.MT5Account, &info.Platform, &info.Description)
		if err != nil {
			return nil, fmt.Errorf("invalid EA key")
		}
		// Update last_used
		db.Exec(`UPDATE investor_ea_keys SET last_used=now() WHERE id=$1`, info.KeyID)
		return &info, nil
	}

	if legacyID != "" {
		// Legacy: X-Investor-ID — lookup or create default key entry
		var info EAKeyInfo
		info.InvestorID = legacyID
		info.Platform = "MT5"
		// Try get mt5_account from investor_settings
		db.QueryRow(
			`SELECT COALESCE(mt5_account,'') FROM investor_settings WHERE investor_id=$1::uuid`,
			legacyID).Scan(&info.MT5Account)
		return &info, nil
	}

	return nil, fmt.Errorf("missing X-EA-Key or X-Investor-ID header")
}

// EAMiddleware — gin middleware untuk EA routes
func EAMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		info, err := lookupEAKey(db, c)
		if err != nil {
			c.JSON(401, gin.H{"ok": false, "error": err.Error()})
			c.Abort()
			return
		}
		c.Set("investor_id", info.InvestorID)
		c.Set("ea_key_id", info.KeyID)
		c.Set("mt5_account", info.MT5Account)
		c.Set("platform", info.Platform)
		c.Next()
	}
}

// getEAInvestorID — helper untuk EA handlers
func getEAInvestorID(c *gin.Context) string {
	if v, ok := c.Get("investor_id"); ok {
		return v.(string)
	}
	// fallback legacy
	return c.GetHeader("X-Investor-ID")
}

// GenerateEAKeyForAccount — POST /api/investor/ea-keys
// Generate EA key untuk akun MT5/MT4 tertentu
func (h *Handler) GenerateEAKeyForAccount(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	var req struct {
		MT5Account  string `json:"mt5Account"`
		Platform    string `json:"platform"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"ok": false, "error": "invalid request"}); return
	}
	if req.Platform == "" { req.Platform = "MT5" }

	// Generate raw key
	rawKey := genKey()
	keyHash := hashEAKeyV2(rawKey)
	keyID := "eakey-" + rawKey[7:15]

	_, err := h.service.repo.DB.Exec(
		`INSERT INTO investor_ea_keys
			(id, investor_id, key_hash, mt5_account, platform, description, created_at)
		VALUES ($1, $2::uuid, $3, $4, $5, $6, now())
		ON CONFLICT (key_hash) DO NOTHING`,
		keyID, uid, keyHash, req.MT5Account, req.Platform, req.Description)
	if err != nil {
		c.JSON(500, gin.H{"ok": false, "error": "key generation failed: " + err.Error()}); return
	}

	c.JSON(200, gin.H{
		"ok": true,
		"eaKey": rawKey,
		"keyId": keyID,
		"platform": req.Platform,
		"mt5Account": req.MT5Account,
		"message": "Save this key — it will not be shown again!",
	})
}

// GetEAKeys — GET /api/investor/ea-keys
// List semua EA keys investor
func (h *Handler) GetEAKeys(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	rows, err := h.service.repo.DB.Query(
		`SELECT id, mt5_account, platform, description,
		       created_at, last_used, equity
		 FROM investor_ea_keys
		 WHERE investor_id = $1::uuid
		 ORDER BY created_at DESC`,
		uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "db error"}); return }
	defer rows.Close()

	type KeyRow struct {
		ID          string  `json:"id"`
		MT5Account  string  `json:"mt5Account"`
		Platform    string  `json:"platform"`
		Description string  `json:"description"`
		CreatedAt   string  `json:"createdAt"`
		LastUsed    string  `json:"lastUsed"`
		Equity      float64 `json:"equity"`
	}
	var keys []KeyRow
	for rows.Next() {
		var k KeyRow
		var createdAt time.Time
		var lastUsed sql.NullTime
		if err := rows.Scan(&k.ID, &k.MT5Account, &k.Platform, &k.Description,
			&createdAt, &lastUsed, &k.Equity); err != nil { continue }
		k.CreatedAt = createdAt.Format("2006-01-02 15:04")
		if lastUsed.Valid { k.LastUsed = lastUsed.Time.Format("2006-01-02 15:04") }
		keys = append(keys, k)
	}
	if keys == nil { keys = []KeyRow{} }
	c.JSON(200, gin.H{"ok": true, "keys": keys})
}

// DeleteEAKey — DELETE /api/investor/ea-keys/:id
func (h *Handler) DeleteEAKey(c *gin.Context) {
	uid, ok := getUID(c)
	if !ok { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }
	keyID := c.Param("id")
	_, err := h.service.repo.DB.Exec(
		`DELETE FROM investor_ea_keys WHERE id=$1 AND investor_id=$2::uuid`,
		keyID, uid)
	if err != nil { c.JSON(500, gin.H{"ok": false, "error": "delete failed"}); return }
	c.JSON(200, gin.H{"ok": true, "message": "EA key deleted"})
}

// EAPushEquityV2 — POST /api/ea/investor/push-equity
// Simpan equity per EA key (per akun MT5)
func (h *Handler) EAPushEquityV2(c *gin.Context) {
	investorID := getEAInvestorID(c)
	if investorID == "" { c.JSON(401, gin.H{"ok": false, "error": "unauthorized"}); return }

	var req struct {
		Equity  float64 `json:"equity"`
		Balance float64 `json:"balance"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Equity <= 0 {
		c.JSON(400, gin.H{"ok": false, "error": "invalid equity"}); return
	}

	keyID, _ := c.Get("ea_key_id")

	// Update equity di investor_ea_keys (per akun)
	if keyID != nil && keyID.(string) != "" {
		h.service.repo.DB.Exec(
			`UPDATE investor_ea_keys SET equity=$1, last_equity_at=now() WHERE id=$2`,
			req.Equity, keyID)
	}

	// Update total investor equity di investor_settings (sum semua akun)
	h.service.repo.DB.Exec(
		`UPDATE investor_settings SET
		  investor_equity = (SELECT COALESCE(SUM(equity),0) FROM investor_ea_keys WHERE investor_id=$1::uuid),
		  updated_at = now()
		WHERE investor_id = $1::uuid`,
		investorID)

	c.JSON(200, gin.H{"ok": true, "equity": req.Equity})
}


