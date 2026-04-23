package admin

import (
"database/sql"
"encoding/csv"
"fmt"
"io"
"strings"
"time"

"github.com/gin-gonic/gin"
)

type SignalImportHandler struct {
DB *sql.DB
}

type ImportedSignal struct {
Row       int    `json:"row"`
Pair      string `json:"pair"`
Direction string `json:"direction"`
Status    string `json:"status"`
Error     string `json:"error,omitempty"`
}

func (h *SignalImportHandler) ImportSignals(c *gin.Context) {
setId := strings.TrimSpace(c.PostForm("set_id"))
if setId == "" {
c.JSON(400, gin.H{"ok": false, "error": "set_id required"})
return
}

var analystId string
err := h.DB.QueryRow(`SELECT analyst_id FROM analyst_signal_sets WHERE id=$1`, setId).Scan(&analystId)
if err != nil {
c.JSON(400, gin.H{"ok": false, "error": "signal set not found"})
return
}

file, _, err := c.Request.FormFile("file")
if err != nil {
c.JSON(400, gin.H{"ok": false, "error": "file required"})
return
}
defer file.Close()

reader := csv.NewReader(file)
reader.TrimLeadingSpace = true

header, err := reader.Read()
if err != nil {
c.JSON(400, gin.H{"ok": false, "error": "cannot read CSV header"})
return
}

colIdx := make(map[string]int)
for i, col := range header {
colIdx[strings.ToLower(strings.TrimSpace(col))] = i
}
required := []string{"pair", "direction", "entry", "sl", "tp", "status"}
for _, r := range required {
if _, ok := colIdx[r]; !ok {
c.JSON(400, gin.H{"ok": false, "error": fmt.Sprintf("missing column: %s", r)})
return
}
}

results := []ImportedSignal{}
imported := 0
skipped := 0
row := 1

for {
record, err := reader.Read()
if err == io.EOF {
break
}
if err != nil {
skipped++
continue
}
row++

get := func(col string) string {
i, ok := colIdx[col]
if !ok || i >= len(record) {
return ""
}
return strings.TrimSpace(record[i])
}

pair := strings.ToUpper(get("pair"))
direction := strings.ToUpper(get("direction"))
entry := get("entry")
sl := get("sl")
tp := get("tp")
status := strings.ToUpper(get("status"))
issuedAt := get("issued_at")
closedAt := get("closed_at")

if pair == "" || entry == "" || sl == "" || tp == "" {
results = append(results, ImportedSignal{Row: row, Pair: pair, Direction: direction, Status: status, Error: "missing required fields"})
skipped++
continue
}
if direction != "BUY" && direction != "SELL" {
results = append(results, ImportedSignal{Row: row, Pair: pair, Direction: direction, Status: status, Error: "direction must be BUY or SELL"})
skipped++
continue
}
validStatus := map[string]bool{"PENDING": true, "RUNNING": true, "CLOSED_TP": true, "CLOSED_SL": true, "CANCELLED_MANUAL": true}
if !validStatus[status] {
status = "CLOSED_TP"
}

layouts := []string{"2006-01-02 15:04:05", "2006-01-02", "02/01/2006", "01/02/2006"}
var issuedAtVal, closedAtVal interface{}
for _, l := range layouts {
if t, err2 := time.Parse(l, issuedAt); err2 == nil {
issuedAtVal = t
break
}
}
for _, l := range layouts {
if t, err2 := time.Parse(l, closedAt); err2 == nil {
closedAtVal = t
break
}
}

_, dbErr := h.DB.Exec(`
INSERT INTO analyst_signals
(analyst_id, set_id, pair, direction, entry, sl, tp, status, issued_at, running_at, closed_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now(),now())
ON CONFLICT (set_id, pair, direction, issued_at) DO UPDATE SET
  entry=EXCLUDED.entry, sl=EXCLUDED.sl, tp=EXCLUDED.tp,
  status=EXCLUDED.status, closed_at=EXCLUDED.closed_at, updated_at=now()`,
analystId, setId, pair, direction, entry, sl, tp, status,
issuedAtVal, issuedAtVal, closedAtVal,
)
if dbErr != nil {
results = append(results, ImportedSignal{Row: row, Pair: pair, Direction: direction, Status: status, Error: dbErr.Error()})
skipped++
continue
}
results = append(results, ImportedSignal{Row: row, Pair: pair, Direction: direction, Status: status})
imported++
}

c.JSON(200, gin.H{
"ok":       true,
"imported": imported,
"skipped":  skipped,
"rows":     results,
})
}

func (h *SignalImportHandler) DownloadTemplate(c *gin.Context) {
c.Header("Content-Disposition", "attachment; filename=signal_import_template.csv")
c.Header("Content-Type", "text/csv")
c.String(200, "pair,direction,entry,sl,tp,status,issued_at,closed_at\nXAUUSD,BUY,2950.00,2930.00,2980.00,CLOSED_TP,2026-01-15 08:00:00,2026-01-15 14:30:00\nEURUSD,SELL,1.0850,1.0900,1.0780,CLOSED_SL,2026-01-16 09:00:00,2026-01-16 11:00:00\n")
}
