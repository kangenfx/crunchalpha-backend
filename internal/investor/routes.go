package investor

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, db *sql.DB) {
	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	// ── Investor dashboard routes ─────────────────────
	r.GET("/portfolio", handler.GetPortfolio)
	r.GET("/allocations", handler.GetAllocations)
	r.POST("/allocations", handler.SetAllocation)
	r.POST("/subscribe", handler.Subscribe)
	r.GET("/subscriptions", handler.GetSubscriptions)
	r.GET("/traders", handler.GetTraderList)

	// ── Settings ──────────────────────────────────────
	r.GET("/settings", handler.GetSettings)
	r.POST("/settings", handler.SaveSettings)
	r.POST("/settings/generate-key", handler.GenerateEAKey)

	// ── Signal orders history ─────────────────────────
	r.GET("/signal-orders", handler.GetSignalOrders)

	// ── Copy trade history ────────────────────────────
	r.GET("/copy-trade-history", handler.GetCopyTradeHistory)
}

func RegisterEARoutes(r *gin.RouterGroup, db *sql.DB) {
	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	// ── EA Investor endpoints ─────────────────────────
	r.GET("/settings", handler.EAGetSettings)
	r.GET("/pending-signals", handler.EAGetPendingSignals)
	r.POST("/order-update", handler.EAOrderUpdate)

	// ── Copy Trade EA endpoints ───────────────────────
	r.POST("/push-equity", handler.EAPushEquity)
	r.GET("/pending-copy-trades", handler.EAGetPendingCopyTrades)
	r.POST("/copy-trade-update", handler.EACopyTradeUpdate)
}
