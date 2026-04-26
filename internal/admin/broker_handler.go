package admin

import (
"database/sql"
"net/http"

"github.com/gin-gonic/gin"
"github.com/lib/pq"
)

type BrokerHandler struct {
DB *sql.DB
}

func NewBrokerHandler(db *sql.DB) *BrokerHandler {
return &BrokerHandler{DB: db}
}

type Broker struct {
ID                 string   `json:"id"`
Name               string   `json:"name"`
LogoURL            *string  `json:"logo_url"`
WebsiteURL         string   `json:"website_url"`
ReferralLink       *string  `json:"referral_link"`
BrokerPaysPerLot   float64  `json:"broker_pays_per_lot"`
MinDeposit         float64  `json:"min_deposit"`
IsRecommended      bool     `json:"is_recommended"`
IsActive           bool     `json:"is_active"`
Description        *string  `json:"description"`
DisplayOrder       int      `json:"display_order"`
Regulation         string   `json:"regulation"`
Leverage           string   `json:"leverage"`
Rating             float64  `json:"rating"`
Headquarters       string   `json:"headquarters"`
HQCountryCode      string   `json:"hq_country_code"`
YearFounded        int      `json:"year_founded"`
TradingInstruments string   `json:"trading_instruments"`
SupportedPlatforms []string `json:"supported_platforms"`
}

func (h *BrokerHandler) ListBrokers(c *gin.Context) {
rows, err := h.DB.Query(`
SELECT id, name, logo_url, website_url, referral_link,
       broker_pays_per_lot, min_deposit, is_recommended, is_active,
       description, display_order,
       COALESCE(regulation,''), COALESCE(leverage,'1:500'), COALESCE(rating,4.0),
       COALESCE(headquarters,''), COALESCE(hq_country_code,''),
       COALESCE(year_founded,0), COALESCE(trading_instruments,''),
       COALESCE(supported_platforms, ARRAY['MT4','MT5'])
FROM ib_brokers ORDER BY display_order, name
`)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
return
}
defer rows.Close()
var brokers []Broker
for rows.Next() {
var b Broker
err := rows.Scan(
&b.ID, &b.Name, &b.LogoURL, &b.WebsiteURL, &b.ReferralLink,
&b.BrokerPaysPerLot, &b.MinDeposit, &b.IsRecommended, &b.IsActive,
&b.Description, &b.DisplayOrder,
&b.Regulation, &b.Leverage, &b.Rating,
&b.Headquarters, &b.HQCountryCode,
&b.YearFounded, &b.TradingInstruments,
pq.Array(&b.SupportedPlatforms),
)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
return
}
brokers = append(brokers, b)
}
if brokers == nil {
brokers = []Broker{}
}
c.JSON(http.StatusOK, gin.H{"data": brokers})
}

func (h *BrokerHandler) CreateBroker(c *gin.Context) {
var req struct {
Name               string  `json:"name" binding:"required"`
LogoURL            *string `json:"logo_url"`
WebsiteURL         string  `json:"website_url" binding:"required"`
ReferralLink       *string `json:"referral_link"`
BrokerPaysPerLot   float64 `json:"broker_pays_per_lot"`
MinDeposit         float64 `json:"min_deposit"`
IsRecommended      bool    `json:"is_recommended"`
IsActive           bool    `json:"is_active"`
Description        *string `json:"description"`
DisplayOrder       int     `json:"display_order"`
Regulation         string  `json:"regulation"`
Leverage           string  `json:"leverage"`
Rating             float64 `json:"rating"`
Headquarters       string  `json:"headquarters"`
HQCountryCode      string  `json:"hq_country_code"`
YearFounded        int     `json:"year_founded"`
TradingInstruments string  `json:"trading_instruments"`
}
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
return
}
if req.Rating == 0 {
req.Rating = 4.0
}
if req.Leverage == "" {
req.Leverage = "1:500"
}
var id string
err := h.DB.QueryRow(`
INSERT INTO ib_brokers (
name, logo_url, website_url, referral_link,
broker_pays_per_lot, min_deposit, is_recommended, is_active,
description, display_order,
regulation, leverage, rating, headquarters, hq_country_code,
year_founded, trading_instruments
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
RETURNING id
`, req.Name, req.LogoURL, req.WebsiteURL, req.ReferralLink,
req.BrokerPaysPerLot, req.MinDeposit, req.IsRecommended, req.IsActive,
req.Description, req.DisplayOrder,
req.Regulation, req.Leverage, req.Rating, req.Headquarters, req.HQCountryCode,
req.YearFounded, req.TradingInstruments,
).Scan(&id)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
return
}
c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Broker created"})
}

func (h *BrokerHandler) UpdateBroker(c *gin.Context) {
id := c.Param("id")
var req struct {
Name               string  `json:"name"`
LogoURL            *string `json:"logo_url"`
WebsiteURL         string  `json:"website_url"`
ReferralLink       *string `json:"referral_link"`
BrokerPaysPerLot   float64 `json:"broker_pays_per_lot"`
MinDeposit         float64 `json:"min_deposit"`
IsRecommended      bool    `json:"is_recommended"`
IsActive           bool    `json:"is_active"`
Description        *string `json:"description"`
DisplayOrder       int     `json:"display_order"`
Regulation         string  `json:"regulation"`
Leverage           string  `json:"leverage"`
Rating             float64 `json:"rating"`
Headquarters       string  `json:"headquarters"`
HQCountryCode      string  `json:"hq_country_code"`
YearFounded        int     `json:"year_founded"`
TradingInstruments string  `json:"trading_instruments"`
}
if err := c.ShouldBindJSON(&req); err != nil {
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
return
}
_, err := h.DB.Exec(`
UPDATE ib_brokers SET
name=$1, logo_url=$2, website_url=$3, referral_link=$4,
broker_pays_per_lot=$5, min_deposit=$6, is_recommended=$7,
is_active=$8, description=$9, display_order=$10,
regulation=$11, leverage=$12, rating=$13,
headquarters=$14, hq_country_code=$15,
year_founded=$16, trading_instruments=$17,
updated_at=NOW()
WHERE id=$18
`, req.Name, req.LogoURL, req.WebsiteURL, req.ReferralLink,
req.BrokerPaysPerLot, req.MinDeposit, req.IsRecommended,
req.IsActive, req.Description, req.DisplayOrder,
req.Regulation, req.Leverage, req.Rating,
req.Headquarters, req.HQCountryCode,
req.YearFounded, req.TradingInstruments, id,
)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
return
}
c.JSON(http.StatusOK, gin.H{"message": "Broker updated"})
}

func (h *BrokerHandler) DeleteBroker(c *gin.Context) {
id := c.Param("id")
_, err := h.DB.Exec(`DELETE FROM ib_brokers WHERE id=$1`, id)
if err != nil {
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
return
}
c.JSON(http.StatusOK, gin.H{"message": "Broker deleted"})
}
