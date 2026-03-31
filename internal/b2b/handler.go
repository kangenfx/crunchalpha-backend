package b2b

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	repo    *Repository
	manager *ManagerConnector
	rest    *RestConnector
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:    repo,
		manager: NewManagerConnector(repo),
		rest:    NewRestConnector(repo),
	}
}

func (h *Handler) RegisterBroker(c *gin.Context) {
	var cfg BrokerConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SaveBrokerConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save broker"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "broker registered successfully", "broker": cfg})
}

func (h *Handler) ListBrokers(c *gin.Context) {
	brokers, err := h.repo.GetActiveBrokers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch brokers"})
		return
	}
	for i := range brokers {
		brokers[i].ManagerPassword = "***"
		brokers[i].RestSecret = "***"
	}
	c.JSON(http.StatusOK, gin.H{"brokers": brokers})
}

func (h *Handler) SyncBroker(c *gin.Context) {
	brokerCode := c.Param("broker_code")
	brokers, err := h.repo.GetActiveBrokers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch brokers"})
		return
	}
	var target *BrokerConfig
	for i := range brokers {
		if brokers[i].BrokerCode == brokerCode {
			target = &brokers[i]
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "broker not found"})
		return
	}
	var result *SyncResult
	var syncErr error
	switch target.IntegrationType {
	case "manager_api":
		result, syncErr = h.manager.Sync(target)
	case "rest_api":
		result, syncErr = h.rest.Sync(target)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported integration type"})
		return
	}
	if syncErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": syncErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sync completed", "result": result})
}

func (h *Handler) RegisterWhiteLabel(c *gin.Context) {
	var wl WhiteLabelConfig
	if err := c.ShouldBindJSON(&wl); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SaveWhiteLabel(&wl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save white label config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "white label configured successfully", "whitelabel": wl})
}

func (h *Handler) GetWhiteLabelConfig(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		domain = c.Request.Host
	}
	wl, err := h.repo.GetWhiteLabelByDomain(domain)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"brand_name":    "CrunchAlpha",
			"logo_url":      "/logo.png",
			"primary_color": "#38BDF8",
			"is_default":    true,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"brand_name":    wl.BrandName,
		"logo_url":      wl.LogoURL,
		"primary_color": wl.PrimaryColor,
		"is_default":    false,
	})
}
