package shared

// UserRole defines user primary role
type UserRole string

const (
	RoleTrader   UserRole = "TRADER"
	RoleInvestor UserRole = "INVESTOR"
	RoleAnalyst  UserRole = "ANALYST"
)

// ViewMode defines lens perspective
type ViewMode string

const (
	ModeA ViewMode = "MODE_A" // Trader lens
	ModeB ViewMode = "MODE_B" // Investor lens
	ModeN ViewMode = "MODE_N" // Analyst lens
)

// User can switch between modes regardless of role
// This is the KEY DIFFERENTIATOR
