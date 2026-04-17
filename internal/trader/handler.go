package trader

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"crunchalpha-v3/internal/alpharank"
)

type Handler struct {
	alpharankService *alpharank.Service
	service *Service
	DB *sql.DB
}

func NewHandler(service *Service, alpharankService *alpharank.Service, db *sql.DB) *Handler {
	return &Handler{
		service:          service,
		alpharankService: alpharankService,
		DB:               db,
	}
}

// GetAccounts returns all trading accounts for current user
func (h *Handler) GetAccounts(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	accounts, err := h.service.GetUserAccounts(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch accounts",
			"message": err.Error(),
		})
		return
	}

	// Return empty array if no accounts
	if accounts == nil {
		accounts = []TraderAccount{}
	}

	c.JSON(http.StatusOK, AccountsResponse{
		Accounts: accounts,
	})
}

// GetDashboard returns complete dashboard data
func (h *Handler) GetDashboard(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Missing parameter",
			"message": "account_id query parameter is required",
		})
		return
	}

	dashboard, err := h.service.GetDashboardWithAlphaRank(accountID, userID.(string))
	if err != nil {
		if err.Error() == "account not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Account not found",
				"message": "The requested account does not exist or does not belong to you",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch dashboard",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dashboard)
}

// CreateDummyAccount creates a test account (development only)
func (h *Handler) CreateDummyAccount(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Nickname string `json:"nickname" binding:"required"`
		Broker   string `json:"broker" binding:"required"`
		Platform string `json:"platform" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	account, err := h.service.CreateDummyAccount(
		userID.(string),
		req.Nickname,
		req.Broker,
		req.Platform,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create account",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, account)
}
// CreateAccount handles new account registration (PRODUCTION GRADE with logging)
func (h *Handler) CreateAccount(c *gin.Context) {
	var payload struct {
		AccountNumber       string `json:"account_number" binding:"required"`
		Broker              string `json:"broker" binding:"required"`
		Platform            string `json:"platform" binding:"required"`
		Server              string `json:"server"`
		InvestorPassword    string `json:"investor_password"`
		Nickname            string `json:"nickname"`
		Currency            string `json:"currency"`
		AccountRole         string `json:"account_role"`
		About               string `json:"about"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("❌ CreateAccount - Bind error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		log.Printf("❌ CreateAccount - No user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if payload.Currency == "" {
		payload.Currency = "USD"
	}
	if payload.AccountRole == "" {
		payload.AccountRole = "trader"
	}

	log.Printf("✅ CreateAccount - Received: account=%s, broker=%s, platform=%s, server=%s, role=%s", 
		payload.AccountNumber, payload.Broker, payload.Platform, payload.Server, payload.AccountRole)

	existing, _ := h.service.GetAccountByNumber(payload.AccountNumber, userID.(string))
	if existing != nil {
		log.Printf("❌ CreateAccount - Account %s already exists", payload.AccountNumber)
		c.JSON(http.StatusConflict, gin.H{"error": "Account already registered"})
		return
	}

	account, err := h.service.CreateAccountFull(
		userID.(string),
		payload.AccountNumber,
		payload.Broker,
		payload.Platform,
		payload.Server,
		payload.InvestorPassword,
		payload.Nickname,
		payload.Currency,
		payload.AccountRole,
		payload.About,
	)
	
	if err != nil {
		log.Printf("❌ CreateAccount - Database error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account", "details": err.Error()})
		return
	}

	// Auto-calculate AlphaRank if account has enough trades
	if err := h.alpharankService.CalculateForAccount(account.ID); err != nil {
		log.Printf("⚠️ CreateAccount - AlphaRank calculation skipped: %v", err)
		// Don't fail account creation, just log warning
	}
	log.Printf("✅ CreateAccount - Success: account_id=%s", account.ID)
	
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Account registered successfully",
		"account": account,
	})
}


// PUT /api/trader/accounts/:id
func (h *Handler) UpdateAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account id required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var payload struct {
		Nickname string `json:"nickname"`
		About    string `json:"about"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateAccountMeta(accountID, userID.(string), payload.Nickname, payload.About); err != nil {
		log.Printf("❌ UpdateAccount error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update account"})
		return
	}

	log.Printf("✅ UpdateAccount - account_id=%s nickname=%s", accountID, payload.Nickname)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Account updated"})
}


// PUT /api/trader/accounts/:id
func (h *Handler) GetAccountSummary(c *gin.Context) {
	userID, _ := c.Get("user_id")
	accountID := c.Query("account_id")
	if accountID == "" { c.JSON(400, gin.H{"error": "account_id required"}); return }
	deposit, withdraw, netProfit, roi := h.service.repo.GetAccountSummary(accountID, userID.(string))
	c.JSON(200, gin.H{
		"ok":            true,
		"totalDeposit":  deposit,
		"totalWithdraw": withdraw,
		"netProfit":     netProfit,
		"roi":           roi,
	})
}

// GET /api/trader/my-followers — investors copying this trader's accounts
func (h *Handler) GetMyFollowers(c *gin.Context) {
	uid, ok := c.Get("user_id")
	if !ok { c.JSON(401, gin.H{"ok":false,"error":"unauthorized"}); return }

	rows, err := h.service.repo.db.Query(`
		SELECT
			cs.id, cs.provider_account_id, ta_p.nickname as provider_nickname,
			cs.follower_account_id, ta_f.nickname as follower_nickname,
			cs.status, cs.lot_calculation_method,
			COALESCE(cs.follower_equity,0) as follower_equity,
			COALESCE(cs.lot_multiplier,1) as lot_multiplier,
			cs.created_at,
			COUNT(ce.id) as total_copies,
			COALESCE(SUM(CASE WHEN ce.success=true THEN 1 ELSE 0 END),0) as wins,
			0::numeric as total_pnl
		FROM copy_subscriptions cs
		JOIN trader_accounts ta_p ON ta_p.id = cs.provider_account_id
		JOIN trader_accounts ta_f ON ta_f.id = cs.follower_account_id
		LEFT JOIN copy_executions ce ON ce.subscription_id = cs.id
		WHERE ta_p.user_id = $1::uuid
		GROUP BY cs.id, cs.provider_account_id, ta_p.nickname,
		         cs.follower_account_id, ta_f.nickname,
		         cs.status, cs.lot_calculation_method,
		         cs.follower_equity, cs.lot_multiplier, cs.created_at
		ORDER BY cs.created_at DESC`, uid)
	if err != nil { c.JSON(500, gin.H{"ok":false,"error":err.Error()}); return }
	defer rows.Close()

	type FollowerRow struct {
		ID                string  `json:"id"`
		ProviderAccountID string  `json:"providerAccountId"`
		ProviderNickname  string  `json:"providerNickname"`
		FollowerAccountID string  `json:"followerAccountId"`
		FollowerNickname  string  `json:"followerNickname"`
		Status            string  `json:"status"`
		LotMethod         string  `json:"lotMethod"`
		FollowerEquity    float64 `json:"followerEquity"`
		LotMultiplier     float64 `json:"lotMultiplier"`
		CreatedAt         string  `json:"createdAt"`
		TotalCopies       int     `json:"totalCopies"`
		Wins              int     `json:"wins"`
		TotalPnl          float64 `json:"totalPnl"`
	}

	var followers []FollowerRow
	for rows.Next() {
		var f FollowerRow
		rows.Scan(&f.ID, &f.ProviderAccountID, &f.ProviderNickname,
			&f.FollowerAccountID, &f.FollowerNickname,
			&f.Status, &f.LotMethod, &f.FollowerEquity, &f.LotMultiplier,
			&f.CreatedAt, &f.TotalCopies, &f.Wins, &f.TotalPnl)
		followers = append(followers, f)
	}
	if followers == nil { followers = []FollowerRow{} }

	// Summary stats
	var totalFollowers, activeFollowers int
	var totalAUM float64
	for _, f := range followers {
		totalFollowers++
		if f.Status == "ACTIVE" { activeFollowers++; totalAUM += f.FollowerEquity }
	}

	c.JSON(200, gin.H{
		"ok": true,
		"followers": followers,
		"summary": gin.H{
			"totalFollowers":  totalFollowers,
			"activeFollowers": activeFollowers,
			"totalAUM":        totalAUM,
		},
	})
}
