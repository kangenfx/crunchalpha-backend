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

layouts := []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "1/2/2006 15:04", "1/2/2006 3:04", "01/02/2006 15:04", "2006-01-02", "02/01/2006", "01/02/2006"}
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

// Build issued_at string for dedup — use entry+direction as fallback if nil
issuedAtStr := ""
if issuedAtVal != nil {
    issuedAtStr = issuedAtVal.(time.Time).Format("2006-01-02 15:04")
} else {
    issuedAtStr = fmt.Sprintf("entry-%s-%s-%s", pair, direction, entry)
}

// Check duplicate first
var dupCount int
h.DB.QueryRow(`SELECT COUNT(*) FROM analyst_signals WHERE set_id=$1 AND pair=$2 AND direction=$3 AND issued_at=$4`,
    setId, pair, direction, issuedAtStr).Scan(&dupCount)
if dupCount > 0 {
    // Update existing
    h.DB.Exec(`UPDATE analyst_signals SET entry=$1, sl=$2, tp=$3, status=$4, closed_at=$5, updated_at=now()
        WHERE set_id=$6 AND pair=$7 AND direction=$8 AND issued_at=$9`,
        entry, sl, tp, status, closedAtVal, setId, pair, direction, issuedAtStr)
    results = append(results, ImportedSignal{Row: row, Pair: pair, Direction: direction, Status: status})
    imported++
    continue
}
_, dbErr := h.DB.Exec(`
INSERT INTO analyst_signals
(analyst_id, set_id, pair, direction, entry, sl, tp, status, issued_at, running_at, closed_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now(),now())`,
analystId, setId, pair, direction, entry, sl, tp, status,
issuedAtStr, issuedAtVal, closedAtVal,
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
c.String(200, "pair,direction,entry,sl,tp,status,issued_at,closed_at\n" +
"# Format issued_at & closed_at: YYYY-MM-DD HH:MM:SS or M/D/YYYY H:MM\n" +
"XAUUSD,BUY,2950.00,2930.00,2980.00,CLOSED_TP,2026-01-15 08:00,2026-01-15 14:30\n" +
"EURUSD,SELL,1.0850,1.0900,1.0780,CLOSED_SL,1/16/2026 9:00,1/16/2026 11:00\n" +
"GBPUSD,BUY,1.2700,1.2650,1.2800,CLOSED_TP,1/17/2026 10:00,1/17/2026 16:00\n")
}

// ── DELETE /api/admin/signal-sets/:id ─────────────────────────────────────────
func (h *SignalImportHandler) DeleteSignalSet(c *gin.Context) {
setId := c.Param("id")
if setId == "" {
c.JSON(400, gin.H{"ok": false, "error": "set_id required"})
return
}

// Check exists
var name string
err := h.DB.QueryRow(`SELECT name FROM analyst_signal_sets WHERE id=$1`, setId).Scan(&name)
if err != nil {
c.JSON(404, gin.H{"ok": false, "error": "signal set not found"})
return
}

// Delete signals first (FK set_id ON DELETE SET NULL — but we want hard delete)
var deletedSignals int64
res, err := h.DB.Exec(`DELETE FROM analyst_signals WHERE set_id=$1`, setId)
if err == nil {
deletedSignals, _ = res.RowsAffected()
}

// Delete subscriptions
h.DB.Exec(`DELETE FROM analyst_subscriptions WHERE set_id=$1`, setId)

// Delete signal set
_, err = h.DB.Exec(`DELETE FROM analyst_signal_sets WHERE id=$1`, setId)
if err != nil {
c.JSON(500, gin.H{"ok": false, "error": fmt.Sprintf("delete failed: %v", err)})
return
}

c.JSON(200, gin.H{
"ok":              true,
"deleted_set":     name,
"deleted_signals": deletedSignals,
})
}

// ── GET /api/admin/signals/export/:setId ──────────────────────────────────────
func (h *SignalImportHandler) ExportSignals(c *gin.Context) {
setId := c.Param("setId")

var setName string
err := h.DB.QueryRow(`SELECT name FROM analyst_signal_sets WHERE id=$1`, setId).Scan(&setName)
if err != nil {
c.JSON(404, gin.H{"ok": false, "error": "signal set not found"})
return
}

rows, err := h.DB.Query(`
SELECT pair, direction, entry, sl, tp, status,
       COALESCE(issued_at, ''),
       COALESCE(closed_at::text, '')
FROM analyst_signals
WHERE set_id=$1
ORDER BY issued_at ASC NULLS LAST, id ASC
`, setId)
if err != nil {
c.JSON(500, gin.H{"ok": false, "error": "db error"})
return
}
defer rows.Close()

filename := fmt.Sprintf("signals_%s.csv", strings.ReplaceAll(setName, " ", "_"))
c.Header("Content-Disposition", "attachment; filename="+filename)
c.Header("Content-Type", "text/csv")

w := csv.NewWriter(c.Writer)
w.Write([]string{"pair", "direction", "entry", "sl", "tp", "status", "issued_at", "closed_at"})

for rows.Next() {
var pair, direction, entry, sl, tp, status, issuedAt, closedAt string
rows.Scan(&pair, &direction, &entry, &sl, &tp, &status, &issuedAt, &closedAt)
w.Write([]string{pair, direction, entry, sl, tp, status, issuedAt, closedAt})
}
w.Flush()
}
