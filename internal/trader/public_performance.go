package trader

import (
	"database/sql"
)

// GetMonthlyPerformanceFromDB - public function for cross-package use (investor lens)
func GetMonthlyPerformanceFromDB(db *sql.DB, accountID string) ([]map[string]interface{}, error) {
	s := &Service{repo: &Repository{db: db}}
	return s.GetMonthlyPerformanceFromDB(accountID)
}

// GetWeeklyPerformanceFromDB - public function for cross-package use (investor lens)
func GetWeeklyPerformanceFromDB(db *sql.DB, accountID string) ([]map[string]interface{}, error) {
	s := &Service{repo: &Repository{db: db}}
	return s.GetWeeklyPerformanceFromDB(accountID)
}
