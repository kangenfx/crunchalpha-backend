package admin

import (
	"database/sql"
	"net/http"
	"github.com/gin-gonic/gin"
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

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	_, err := h.DB.Exec(`DELETE FROM users WHERE id=$1`, id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
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
