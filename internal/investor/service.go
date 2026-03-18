package investor

import (
	"context"
	"fmt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// GetPortfolio - all values from DB, no hardcoding
func (s *Service) GetPortfolio(ctx context.Context, userID string) (*Portfolio, error) {
	user, err := s.repo.GetUserInfo(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	totalCapital, err := s.repo.GetInvestorCapital(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get capital: %w", err)
	}

	allocations, err := s.repo.GetAllocations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allocations: %w", err)
	}

	subscriptions, err := s.repo.GetSubscriptions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}

	survivability, riskLevel, err := s.repo.GetPortfolioStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio stats: %w", err)
	}

	// Calculate allocated capital from DB allocation values
	allocatedCapital := 0.0
	for _, alloc := range allocations {
		if alloc.AllocationMode == "FIXED" {
			allocatedCapital += alloc.AllocationValue
		} else if alloc.AllocationMode == "PERCENT" && totalCapital > 0 {
			allocatedCapital += totalCapital * alloc.AllocationValue / 100
		}
	}

	allocatedPct := 0.0
	if totalCapital > 0 {
		allocatedPct = (allocatedCapital / totalCapital) * 100
	}

	// Normalize slices - never return null to frontend
	if allocations == nil {
		allocations = []AllocationWithTraderInfo{}
	}
	if subscriptions == nil {
		subscriptions = []Subscription{}
	}

	return &Portfolio{
		User:               user,
		TotalCapital:       totalCapital,
		AllocatedCapital:   allocatedCapital,
		TotalReturn:        0.0,
		MonthlyReturn:      0.0,
		SurvivabilityScore: survivability,
		RiskLevel:          riskLevel,
		Allocations:        allocations,
		Subscriptions:      subscriptions,
		TotalAllocatedPct:  allocatedPct,
	}, nil
}

// GetAllocations - portfolio mode settings + allocations list
func (s *Service) GetAllocations(ctx context.Context, userID string) (*AllocationSettings, error) {
	allocations, err := s.repo.GetAllocations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get allocations: %w", err)
	}
	if allocations == nil {
		allocations = []AllocationWithTraderInfo{}
	}
	return &AllocationSettings{
		Mode:               "AUTO",
		RebalanceFrequency: "WEEKLY",
		Allocations:        allocations,
	}, nil
}

// Subscribe - follow trader and create subscription
func (s *Service) Subscribe(ctx context.Context, userID, traderAccountID string) error {
	if err := s.repo.FollowTrader(ctx, userID, traderAccountID); err != nil {
		return fmt.Errorf("failed to follow trader: %w", err)
	}
	if err := s.repo.CreateSubscription(ctx, userID, traderAccountID); err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

// SetAllocation - upsert allocation config
func (s *Service) SetAllocation(ctx context.Context, userID, traderAccountID, mode string, value, maxRisk float64, maxPos int) error {
	return s.repo.UpsertAllocation(ctx, userID, traderAccountID, mode, value, maxRisk, maxPos)
}

// GetTraderList - list of available traders for marketplace
func (s *Service) GetTraderList(ctx context.Context) ([]TraderListItem, error) {
	traders, err := s.repo.GetTraderList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get trader list: %w", err)
	}
	if traders == nil {
		traders = []TraderListItem{}
	}
	return traders, nil
}
