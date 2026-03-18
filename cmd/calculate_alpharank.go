package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"crunchalpha-v3/internal/alpharank"
)

func main() {
	// Connect via localhost (Docker port forwarding)
	connStr := "host=localhost port=5432 user=alpha_user password=alpha_password dbname=crunchalpha sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping:", err)
	}

	accountID := "0f89e77e-5bfd-444d-be1d-3dca6cec5108"

	fmt.Println("========================================")
	fmt.Println("CALCULATING ALPHARANK FOR ACCOUNT")
	fmt.Println("Account ID:", accountID)
	fmt.Println("========================================")

	service := alpharank.NewService(db)
	err = service.CalculateForAccount(accountID)
	if err != nil {
		log.Fatal("Calculation failed:", err)
	}

	fmt.Println("✅ AlphaRank calculated successfully!")
	fmt.Println("")

	var (
		p1, p2, p3, p4, p5, p6, p7 float64
		total float64
		grade, badge string
		tradeCount int
	)

	err = db.QueryRow(`
		SELECT 
			profitability_score, risk_score, consistency_score,
			stability_score, activity_score, duration_score, drawdown_score,
			alpha_score, grade, badge, trade_count
		FROM alpha_ranks
		WHERE account_id = $1 AND symbol = 'ALL'
	`, accountID).Scan(&p1, &p2, &p3, &p4, &p5, &p6, &p7, &total, &grade, &badge, &tradeCount)

	if err != nil {
		log.Fatal("Failed to retrieve:", err)
	}

	fmt.Println("========================================")
	fmt.Println("ALPHARANK RESULTS")
	fmt.Println("========================================")
	fmt.Printf("Trades Analyzed: %d\n", tradeCount)
	fmt.Printf("P1 Profitability: %.2f\n", p1)
	fmt.Printf("P2 Risk:          %.2f\n", p2)
	fmt.Printf("P3 Consistency:   %.2f\n", p3)
	fmt.Printf("P4 Recovery:      %.2f\n", p4)
	fmt.Printf("P5 Edge:          %.2f\n", p5)
	fmt.Printf("P6 Discipline:    %.2f\n", p6)
	fmt.Printf("P7 Track Record:  %.2f\n", p7)
	fmt.Println("========================================")
	fmt.Printf("Total Score:      %.2f\n", total)
	fmt.Printf("Grade:            %s\n", grade)
	fmt.Printf("Badge:            %s\n", badge)
	fmt.Println("========================================")
}
