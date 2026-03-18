package main

import (
        "crunchalpha-v3/internal/alpharank"
        "crunchalpha-v3/internal/apikey"
        "crunchalpha-v3/internal/ea"
        "crunchalpha-v3/internal/investor"
	"crunchalpha-v3/internal/analyst"
        "log"
        "time"

        "crunchalpha-v3/internal/database"
        "crunchalpha-v3/internal/auth"
        "crunchalpha-v3/internal/profile"
        "crunchalpha-v3/internal/trader"
        "crunchalpha-v3/internal/ratelimit"
        "crunchalpha-v3/internal/admin"
	"crunchalpha-v3/internal/middleware"

        "github.com/gin-gonic/gin"
        "github.com/gin-contrib/cors"
)

func main() {
        db, err := database.Connect()
        if err != nil {
                log.Fatal("Database connection failed:", err)
        }
        defer db.Close()

        // Initialize services
        authService := auth.NewService(db)
        authHandler := auth.NewHandler(authService)

        profileService := profile.NewService(db)
        profileHandler := profile.NewHandler(profileService)
	// AlphaRank Service
	alpharankService := alpharank.NewService(db)

        traderService := trader.NewService(db)
        traderHandler := trader.NewHandler(traderService, alpharankService)

        // API Key Handler
        apikeyRepo := apikey.NewRepository(db)
        apikeyHandler := apikey.NewHandler(apikeyRepo)

        // EA Handler
        eaRepo := ea.NewRepository(db)
        eaHandler := ea.NewHandler(eaRepo)

        alpharankHandler := alpharank.NewHandler(alpharankService)
        
        // Investor Handler

	// Analyst Handler
	analystHandler := &analyst.Handler{DB: db}

	// Admin Handlers
	adminUserHandler := admin.NewUserHandler(db)
	adminBrokerHandler := admin.NewBrokerHandler(db)
	adminFeeHandler := admin.NewFeeHandler(db)
	adminCashflowHandler := admin.NewCashflowHandler(db)
	adminConfigHandler := admin.NewConfigHandler(db)
	adminFeeCalc := admin.NewFeeCalculator(db)

	// Admin Handlers
        investorRepo := investor.NewRepository(db)
        investorService := investor.NewService(investorRepo)
        investorHandler := investor.NewHandler(investorService)

        rateLimiter := ratelimit.NewLimiter(db)

        r := gin.Default()

        r.Use(cors.New(cors.Config{
                AllowOrigins:     []string{"http://45.32.118.117:5176", "http://45.32.118.117:5177", "http://localhost:5176", "http://localhost:5177"},
                AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
                AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
                AllowCredentials: true,
                MaxAge:           12 * time.Hour,
        }))

        r.GET("/health", func(c *gin.Context) {
                c.JSON(200, gin.H{
                        "status":   "ok",
                        "service":  "crunchalpha-v3",
                        "database": "connected",
                })
        })

        // Rate limit configs
        loginLimit := ratelimit.Config{MaxAttempts: 5, Window: 5 * time.Minute}
        registerLimit := ratelimit.Config{MaxAttempts: 3, Window: 1 * time.Hour}

        // Public auth routes
        apiAuth := r.Group("/api/auth")
        {
                apiAuth.POST("/register", rateLimiter.RateLimitMiddleware("register", registerLimit), authHandler.Register)
                apiAuth.POST("/login", rateLimiter.RateLimitMiddleware("login", loginLimit), authHandler.Login)
                apiAuth.POST("/refresh", authHandler.Refresh)
                apiAuth.POST("/logout", authHandler.Logout)

                apiAuth.POST("/forgot-password", authHandler.ForgotPassword)
                apiAuth.GET("/reset-password/:token", authHandler.ValidateResetToken)
                apiAuth.POST("/reset-password", authHandler.ResetPassword)

                apiAuth.GET("/verify-email", authHandler.VerifyEmail)
                apiAuth.POST("/verify-email", authHandler.VerifyEmail)
                apiAuth.POST("/resend-verification", authHandler.ResendVerification)
        }

        // Auth middleware
        authMiddleware := middleware.AuthRequired()

        // Protected profile routes
        apiProfile := r.Group("/api/profile")
        apiProfile.Use(authMiddleware)
        {
                apiProfile.GET("", profileHandler.GetProfile)
                apiProfile.PUT("", profileHandler.UpdateProfile)
        }

        // Protected auth routes
        apiAuthProtected := r.Group("/api/auth")
        apiAuthProtected.Use(authMiddleware)
        {
                apiAuthProtected.POST("/logout-all", authHandler.LogoutAll)
        }

        // Trader routes
        traderRoutes := r.Group("/api/trader")
        traderRoutes.Use(authMiddleware)
        {
                traderRoutes.GET("/accounts", traderHandler.GetAccounts)
                traderRoutes.POST("/accounts", traderHandler.CreateAccount)
                traderRoutes.GET("/dashboard", traderHandler.GetDashboard)
                traderRoutes.POST("/dummy-account", traderHandler.CreateDummyAccount)

                // Dashboard with real AlphaRank from database
                traderRoutes.GET("/dashboard-db", func(c *gin.Context) {
                        accountID := c.Query("account_id")
                        if accountID == "" {
                                c.JSON(400, gin.H{"error": "account_id required"})
                                return
                        }

                        userID := c.GetString("user_id")
                        dashboard, err := traderService.GetDashboardWithAlphaRank(accountID, userID)
                        if err != nil {
                                c.JSON(500, gin.H{"error": err.Error()})
                                return
                        }

                        c.JSON(200, dashboard)
                })
                traderRoutes.GET("/alpharank-perpair", traderHandler.GetAlphaRankPerPair)
                traderRoutes.GET("/trades", traderHandler.GetTrades)
                traderRoutes.GET("/monthly-performance", traderHandler.GetMonthlyPerformance)
                traderRoutes.GET("/weekly-performance", traderHandler.GetWeeklyPerformance)
			traderRoutes.GET("/account-summary", traderHandler.GetAccountSummary)
			traderRoutes.GET("/my-followers", traderHandler.GetMyFollowers)
                traderRoutes.DELETE("/accounts/:account_id", traderHandler.DeleteAccountHandler)
		// Recalculate all accounts for current user
		traderRoutes.POST("/recalculate-all", func(c *gin.Context) {
			userID := c.GetString("user_id")
			rows, err := db.Query("SELECT id FROM trader_accounts WHERE user_id = $1", userID)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			defer rows.Close()


			var accountIDs []string
			for rows.Next() {
				var id string
				rows.Scan(&id)
				accountIDs = append(accountIDs, id)
			}


			// Calculate each account
			success := 0
			failed := 0
			for _, accountID := range accountIDs {
				err := alpharankService.CalculateForAccount(accountID)
				if err != nil {
					failed++
				} else {
					success++
				}
			}


			c.JSON(200, gin.H{
				"success": success,
				"failed":  failed,
				"total":   len(accountIDs),
			})
		})
                traderRoutes.GET("/debug-metrics", func(c *gin.Context) {
                        accountID := c.Query("account_id")
                        if accountID == "" {
                                c.JSON(400, gin.H{"error": "account_id required"})
                                return
                        }
                        userID := c.GetString("user_id")
                        resp, err := traderService.GetDebugMetrics(accountID, userID)
                        if err != nil {
                                c.JSON(500, gin.H{"error": err.Error()})
                                return
                        }
                        c.JSON(200, resp)
                })
        }

        // API Key Management Routes (JWT protected)
        apikeyRoutes := r.Group("/api/apikeys")
        apikeyRoutes.Use(authMiddleware)
        {
                apikeyRoutes.POST("", apikeyHandler.CreateAPIKey)
                apikeyRoutes.GET("", apikeyHandler.ListAPIKeys)
                apikeyRoutes.DELETE("/:key_id", apikeyHandler.RevokeAPIKey)
        }

        // EA Routes - Using API Key Authentication
        // Public marketplace (no auth)
        r.GET("/api/public/marketplace", analystHandler.PublicMarketplace)
	r.GET("/api/public/traders", alpharankHandler.GetPublicTraders)
	r.GET("/api/public/trader/:id", alpharankHandler.GetPublicTraderDetail)
	r.GET("/api/public/analyst-profile/:setId", analystHandler.GetPublicAnalystProfile)

        // EA Download (public, no auth)
        r.GET("/api/ea/download/mt4", func(c *gin.Context) {
                c.FileAttachment("/app/ea/CrunchAlpha_Publisher_MT4.ex4", "CrunchAlpha_Publisher_MT4.ex4")
        })
        r.GET("/api/ea/download/mt5", func(c *gin.Context) {
                c.FileAttachment("/app/ea/CrunchAlpha_Publisher_MT5.ex5", "CrunchAlpha_Publisher_MT5.ex5")
        })

        eaRoutesAPIKey := r.Group("/api/ea")
        eaRoutesAPIKey.Use(middleware.APIKeyAuth(apikeyRepo))
        {
                eaRoutesAPIKey.POST("/trade", eaHandler.ReceiveTrade)
                eaRoutesAPIKey.POST("/account", eaHandler.ReceiveAccount)
                eaRoutesAPIKey.POST("/sync", eaHandler.SyncAccount)
        }

        // Analyst EA Price Feed routes (X-EA-Key auth)
        analystEARoutes := r.Group("/api/ea/analyst")
        analystEARoutes.Use(analystHandler.EAAuthMiddleware())
        {
                analystEARoutes.GET("/pending-signals", analystHandler.EAGetSignals)
                analystEARoutes.POST("/update-signal", analystHandler.EAUpdateSignal)
                analystEARoutes.POST("/batch-update", analystHandler.EABatchUpdate)
        }

        // Investor routes
        investorRoutes := r.Group("/api/investor")
        investorRoutes.Use(authMiddleware)
        {
                investorRoutes.GET("/portfolio", investorHandler.GetPortfolio)
                investorRoutes.GET("/allocations", investorHandler.GetAllocations)
                investorRoutes.POST("/allocations", investorHandler.SetAllocation)
                investorRoutes.POST("/subscribe", investorHandler.Subscribe)
                investorRoutes.GET("/subscriptions", investorHandler.GetSubscriptions)
		investorRoutes.GET("/traders", investorHandler.GetTraderList)
		// Analyst signal set routes
		investorRoutes.GET("/analyst-sets", investorHandler.GetAnalystSets)
		investorRoutes.GET("/analyst-subscriptions", investorHandler.GetAnalystSubscriptions)
		investorRoutes.POST("/analyst-subscribe", investorHandler.SubscribeAnalystSet)
		investorRoutes.POST("/analyst-unsubscribe", investorHandler.UnsubscribeAnalystSet)
			investorRoutes.PUT("/analyst-subscription/:id/mode", investorHandler.UpdateSubscriptionMode)
		investorRoutes.GET("/analyst-feed", investorHandler.GetAnalystFeed)
		investorRoutes.GET("/signal-orders", investorHandler.GetSignalOrders)
		investorRoutes.GET("/settings", investorHandler.GetSettings)
		investorRoutes.GET("/subscription-history", investorHandler.GetSubscriptionHistory)
		investorRoutes.POST("/copy-trader-subscribe", investorHandler.CopyTraderSubscribe)
		investorRoutes.POST("/copy-trader-unsubscribe", investorHandler.CopyTraderUnsubscribe)
		investorRoutes.GET("/copy-trader-subscriptions", investorHandler.GetCopyTraderSubscriptions)
		investorRoutes.GET("/trader-profile/:account_id", investorHandler.GetTraderProfile)
		investorRoutes.GET("/trader-trades", investorHandler.GetTraderTrades)
			investorRoutes.GET("/trade-copies", investorHandler.GetTradeCopies)
			investorRoutes.GET("/performance-overview", investorHandler.GetPerformanceOverview)
			investorRoutes.GET("/affiliate/overview", investorHandler.GetAffiliateOverview)
			investorRoutes.GET("/affiliate/referrals", investorHandler.GetAffiliateReferrals)
			investorRoutes.GET("/affiliate/payouts", investorHandler.GetAffiliatePayouts)
			investorRoutes.POST("/affiliate/recalc", investorHandler.RecalcAffiliateTiers)
			investorRoutes.POST("/affiliate/calculate-payout", investorHandler.CalculateAffiliatePayout)
		investorRoutes.GET("/trader-monthly-performance", investorHandler.GetTraderMonthlyPerformance)
		investorRoutes.GET("/trader-weekly-performance", investorHandler.GetTraderWeeklyPerformance)
		investorRoutes.POST("/settings", investorHandler.SaveSettings)
		investorRoutes.POST("/settings/generate-key", investorHandler.GenerateEAKey)
        }

        // EA Investor routes
        eaInvestorRoutes := r.Group("/api/ea/investor")
        eaInvestorRoutes.GET("/pending-signals", investorHandler.EAGetPendingSignals)
        eaInvestorRoutes.POST("/order-update", investorHandler.EAOrderUpdate)
        eaInvestorRoutes.GET("/settings", investorHandler.EAGetSettings)

        // Analyst routes
	analystRoutes := r.Group("/api/analyst")
	analystRoutes.Use(authMiddleware)
	{
		analystRoutes.GET("/dashboard", analystHandler.Dashboard)
		analystRoutes.GET("/signal-sets", analystHandler.ListSignalSets)
		analystRoutes.POST("/signal-sets", analystHandler.CreateSignalSet)
		analystRoutes.GET("/signals", analystHandler.ListSignals)
		analystRoutes.POST("/signals", analystHandler.CreateSignal)
		analystRoutes.POST("/signals/:id/cancel", analystHandler.CancelSignal)
		analystRoutes.PUT("/signals/:id", analystHandler.UpdateSignal)
		analystRoutes.PUT("/signal-sets/:id", analystHandler.UpdateSignalSet)
		analystRoutes.GET("/performance", analystHandler.Performance)
		analystRoutes.GET("/alpharank", analystHandler.AnalystAlphaRank)
			analystRoutes.GET("/my-subscribers", analystHandler.GetMySubscribers)
	}

	// AlphaRank calculation endpoint
        r.POST("/api/alpharank/calculate/:account_id", alpharankHandler.CalculateAlphaRank)
        r.GET("/api/alpharank/details/:account_id", alpharankHandler.GetDetailedAlphaRank)

        // Admin routes (require is_admin=true via JWT)
	adminRoutes := r.Group("/api/admin")
	adminRoutes.Use(authMiddleware)
	{
		adminRoutes.GET("/stats", adminUserHandler.PlatformStats)
		adminRoutes.GET("/users", adminUserHandler.ListUsers)
		adminRoutes.PUT("/users/:id", adminUserHandler.UpdateUser)
		adminRoutes.DELETE("/users/:id", adminUserHandler.DeleteUser)
		adminRoutes.GET("/trading-accounts", adminUserHandler.ListTradingAccounts)
		adminRoutes.GET("/signal-sets", adminUserHandler.ListSignalSets)
		adminRoutes.GET("/audit-logs", adminUserHandler.AuditLogs)
		adminRoutes.GET("/brokers", adminBrokerHandler.ListBrokers)
		adminRoutes.POST("/brokers", adminBrokerHandler.CreateBroker)
		adminRoutes.PUT("/brokers/:id", adminBrokerHandler.UpdateBroker)
		adminRoutes.DELETE("/brokers/:id", adminBrokerHandler.DeleteBroker)
		adminRoutes.GET("/fee-overrides", adminFeeHandler.ListFeeOverrides)
		adminRoutes.POST("/fee-overrides", adminFeeHandler.UpsertFeeOverride)
		adminRoutes.DELETE("/fee-overrides/:id", adminFeeHandler.DeleteFeeOverride)
		adminRoutes.GET("/default-fees", adminFeeHandler.GetDefaultFees)
		adminRoutes.POST("/recalc-alpharank", analystHandler.AdminRecalcAllAlphaRank)
		adminRoutes.GET("/cashflow/summary", adminCashflowHandler.Summary)
		adminRoutes.GET("/cashflow/signal-subscriptions", adminCashflowHandler.SignalSubscriptions)
		adminRoutes.GET("/cashflow/copy-subscriptions", adminCashflowHandler.CopySubscriptions)
		adminRoutes.GET("/cashflow/user-growth", adminCashflowHandler.UserGrowth)
		adminRoutes.GET("/config/fees", adminConfigHandler.ListFeeConfig)
		adminRoutes.PUT("/config/fees/:key", adminConfigHandler.UpdateFeeConfig)
		adminRoutes.GET("/config/affiliate-tiers", adminConfigHandler.ListAffiliateTiers)
		adminRoutes.PUT("/config/affiliate-tiers/:id", adminConfigHandler.UpdateAffiliateTier)
		adminRoutes.GET("/config/docs", adminConfigHandler.GetDocs)
		adminRoutes.GET("/fee-simulation", adminFeeCalc.SimulateFees)
		adminRoutes.GET("/fee-simulation/all", adminFeeCalc.SimulateAllAccounts)
		adminRoutes.PUT("/trading-accounts/:id/ib-status", adminFeeCalc.UpdateIBStatus)
	}


	log.Println("🚀 CrunchAlpha V3 Server starting on :8090")
        log.Println("📊 AlphaRank™ Engine: ACTIVE")
        log.Println("🗄️  Database: Connected")
        log.Println("💼 Investor Module: ACTIVE")
        r.Run(":8090")
}
