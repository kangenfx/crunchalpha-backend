package investor

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, db *sql.DB) {
	repo := NewRepository(db)
	service := NewService(repo)
	handler := NewHandler(service)

	r.GET("/portfolio", handler.GetPortfolio)
	r.GET("/allocations", handler.GetAllocations)
	r.POST("/allocations", handler.SetAllocation)
	r.POST("/subscribe", handler.Subscribe)
	r.GET("/subscriptions", handler.GetSubscriptions)
	r.GET("/traders", handler.GetTraderList)
}
