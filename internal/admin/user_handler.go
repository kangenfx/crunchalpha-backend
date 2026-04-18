package admin

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	DB *sql.DB
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{DB: db}
}

type AdminUser struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	Name          *string `json:"name"`
	PrimaryRole   string  `json:"primary_role"`
	Status        string  `json:"status"`
	EmailVerified bool    `json:"email_verified"`
	IsAdmin       bool    `json:"is_admin"`
	Country       *string `json:"country"`
	CreatedAt     string  `json:"created_at"`
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, email, name, primary_role, status, email_verified, is_admin, country, created_at
		FROM users ORDER BY created_at DESC
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()
	var users []AdminUser
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.PrimaryRole, &u.Status, &u.EmailVerified, &u.IsAdmin, &u.Country, &u.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
		}
		users = append(users, u)
	}
	if users == nil { users = []AdminUser{} }
	c.JSON(http.StatusOK, gin.H{"data": users})
}

// POST /api/admin/users — create user, bypass email verification
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	if req.Role == "" { req.Role = "trader" }
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"}); return }
	_, err = h.DB.Exec(`
		INSERT INTO users (email, password_hash, name, primary_role, status, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'active', true, NOW(), NOW())
	`, req.Email, string(hash), req.Name, req.Role)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed: " + err.Error()}); return }
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "User created"})
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name        *string `json:"name"`
		PrimaryRole *string `json:"primary_role"`
		Status      *string `json:"status"`
		IsAdmin     *bool   `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	_, err := h.DB.Exec(`UPDATE users SET name=COALESCE($1,name), primary_role=COALESCE($2,primary_role), status=COALESCE($3,status), is_admin=COALESCE($4,is_admin), updated_at=NOW() WHERE id=$5`,
		req.Name, req.PrimaryRole, req.Status, req.IsAdmin, id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

// logAudit writes an audit log entry
func (h *UserHandler) logAudit(c *gin.Context, adminID, eventType, eventAction, targetID, note string) {
	ip := c.ClientIP()
	ua := c.Request.UserAgent()
	meta := `{"target_id":"` + targetID + `","note":"` + note + `"}`
	h.DB.Exec(`
		INSERT INTO audit_logs (id, user_id, event_type, event_action, ip_address, user_agent, metadata, status, created_at)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3, $4, $5, $6::jsonb, 'success', NOW())
	`, adminID, eventType, eventAction, ip, ua, meta)
}

// POST /api/admin/users/:id/verify — force verify email
func (h *UserHandler) ForceVerifyEmail(c *gin.Context) {
	id := c.Param("id")
	_, err := h.DB.Exec(`UPDATE users SET email_verified=true, updated_at=NOW() WHERE id=$1`, id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	adminID, _ := c.Get("user_id")
	h.logAudit(c, adminID.(string), "user", "force_verify_email", id, "")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Email verified"})
}

// POST /api/admin/users/:id/reset-password — admin reset password
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"}); return }
	_, err = h.DB.Exec(`UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2`, string(hash), id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	adminID, _ := c.Get("user_id")
	h.logAudit(c, adminID.(string), "user", "reset_password", id, "")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Password reset"})
}

// POST /api/admin/users/:id/suspend — suspend or unsuspend user
func (h *UserHandler) SuspendUser(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Suspend bool `json:"suspend"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	status := "active"
	if req.Suspend { status = "suspended" }
	_, err := h.DB.Exec(`UPDATE users SET status=$1, updated_at=NOW() WHERE id=$2`, status, id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	adminID, _ := c.Get("user_id")
	h.logAudit(c, adminID.(string), "user", "suspend_user", id, status)
	c.JSON(http.StatusOK, gin.H{"success": true, "status": status})
}

// POST /api/admin/users/:id/impersonate — generate short-lived impersonate token
func (h *UserHandler) ImpersonateUser(c *gin.Context) {
	id := c.Param("id")
	// Check user exists
	var email string
	err := h.DB.QueryRow(`SELECT email FROM users WHERE id=$1`, id).Scan(&email)
	if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "user not found"}); return }
	// Generate random token
	b := make([]byte, 32)
	rand.Read(b)
	token := hex.EncodeToString(b)
	// Store in DB with 15min expiry
	_, err = h.DB.Exec(`
		INSERT INTO impersonate_tokens (token, user_id, expires_at, created_at)
		VALUES ($1, $2::uuid, NOW() + INTERVAL '15 minutes', NOW())
	`, token, id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create token: " + err.Error()}); return }
	adminID, _ := c.Get("user_id")
	h.logAudit(c, adminID.(string), "user", "impersonate", id, email)
	c.JSON(http.StatusOK, gin.H{"success": true, "token": token, "email": email, "expires_in": "15 minutes"})
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	_, err := h.DB.Exec(`DELETE FROM users WHERE id=$1`, id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	adminID, _ := c.Get("user_id")
	h.logAudit(c, adminID.(string), "user", "delete_user", id, "")
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

// DELETE /api/admin/trading-accounts/:id
func (h *UserHandler) DeleteTradingAccount(c *gin.Context) {
	id := c.Param("id")
	_, err := h.DB.Exec(`UPDATE trader_accounts SET status='disabled' WHERE id=$1`, id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Account disabled"})
}

func (h *UserHandler) ListTradingAccounts(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT ta.id, ta.account_number, ta.broker, ta.platform,
		       ta.balance, ta.equity, ta.status, ta.created_at,
		       u.email, u.name, ta.uses_platform_ib,
		       COALESCE(ib.name, '') as ib_broker_name
		FROM trader_accounts ta
		LEFT JOIN users u ON u.id = ta.user_id
		LEFT JOIN ib_brokers ib ON ib.id = ta.ib_broker_id
		ORDER BY ta.created_at DESC
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()
	type Row struct {
		ID             string   `json:"id"`
		AccountNumber  string   `json:"account_number"`
		Broker         string   `json:"broker"`
		Platform       string   `json:"platform"`
		Balance        *float64 `json:"balance"`
		Equity         *float64 `json:"equity"`
		Status         string   `json:"status"`
		CreatedAt      string   `json:"created_at"`
		UserEmail      *string  `json:"user_email"`
		UserName       *string  `json:"user_name"`
		UsesPlatformIB bool     `json:"uses_platform_ib"`
		IBBrokerName   string   `json:"ib_broker_name"`
	}
	var accounts []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.AccountNumber, &r.Broker, &r.Platform, &r.Balance, &r.Equity, &r.Status, &r.CreatedAt, &r.UserEmail, &r.UserName, &r.UsesPlatformIB, &r.IBBrokerName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
		}
		accounts = append(accounts, r)
	}
	if accounts == nil { accounts = []Row{} }
	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

func (h *UserHandler) ListSignalSets(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT s.id, s.name, s.status, s.total_signals,
		       s.win_rate, s.profit_factor, s.created_at,
		       u.email, u.name
		FROM analyst_signal_sets s
		LEFT JOIN users u ON u.id::text = s.analyst_id
		ORDER BY s.created_at DESC
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()
	type Row struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		Status       string   `json:"status"`
		TotalSignals int      `json:"subscribers_count"`
		WinRate      *float64 `json:"win_rate"`
		ProfitFactor *float64 `json:"profit_factor"`
		CreatedAt    string   `json:"created_at"`
		AnalystEmail *string  `json:"analyst_email"`
		AnalystName  *string  `json:"analyst_name"`
	}
	var sets []Row
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.TotalSignals, &r.WinRate, &r.ProfitFactor, &r.CreatedAt, &r.AnalystEmail, &r.AnalystName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
		}
		sets = append(sets, r)
	}
	if sets == nil { sets = []Row{} }
	c.JSON(http.StatusOK, gin.H{"data": sets})
}

func (h *UserHandler) PlatformStats(c *gin.Context) {
	var totalUsers, traders, analysts, investors, totalAccounts, totalSignalSets, totalSignals, totalBrokers int
	h.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role='trader'`).Scan(&traders)
	h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role='analyst'`).Scan(&analysts)
	h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE primary_role='investor'`).Scan(&investors)
	h.DB.QueryRow(`SELECT COUNT(*) FROM trader_accounts`).Scan(&totalAccounts)
	h.DB.QueryRow(`SELECT COUNT(*) FROM analyst_signal_sets`).Scan(&totalSignalSets)
	h.DB.QueryRow(`SELECT COUNT(*) FROM analyst_signals`).Scan(&totalSignals)
	h.DB.QueryRow(`SELECT COUNT(*) FROM ib_brokers WHERE is_active=true`).Scan(&totalBrokers)
	c.JSON(http.StatusOK, gin.H{
		"total_users": totalUsers, "traders": traders, "analysts": analysts, "investors": investors,
		"total_accounts": totalAccounts, "total_signal_sets": totalSignalSets,
		"total_signals": totalSignals, "active_brokers": totalBrokers,
	})
}

func (h *UserHandler) AuditLogs(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT al.id, al.event_type, al.event_action, al.status,
		       al.created_at, u.email
		FROM audit_logs al
		LEFT JOIN users u ON u.id = al.user_id
		ORDER BY al.created_at DESC
		LIMIT 200
	`)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	defer rows.Close()
	type Row struct {
		ID          string  `json:"id"`
		EventType   string  `json:"action"`
		EventAction string  `json:"entity_type"`
		Status      string  `json:"entity_id"`
		CreatedAt   string  `json:"created_at"`
		UserEmail   *string `json:"user_email"`
	}
	var logs []Row
	for rows.Next() {
		var r Row
		rows.Scan(&r.ID, &r.EventType, &r.EventAction, &r.Status, &r.CreatedAt, &r.UserEmail)
		logs = append(logs, r)
	}
	if logs == nil { logs = []Row{} }
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

func (h *UserHandler) UpdateAccountCurrency(c *gin.Context) {
id := c.Param("id")
var body struct {
Currency string `json:"currency"`
}
if err := c.ShouldBindJSON(&body); err != nil {
c.JSON(400, gin.H{"error": "invalid body"}); return
}
if body.Currency != "USD" && body.Currency != "USC" {
c.JSON(400, gin.H{"error": "currency must be USD or USC"}); return
}
_, err := h.DB.Exec(`UPDATE trader_accounts SET currency=$1 WHERE id=$2`, body.Currency, id)
if err != nil {
c.JSON(500, gin.H{"error": err.Error()}); return
}
c.JSON(200, gin.H{"ok": true, "currency": body.Currency})
}
