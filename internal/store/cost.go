package store

import (
	"github.com/user/telegram-claude-bot/internal/costs"
)

type CostRecord = costs.CostRecord

var (
	AddCostRecord    = costs.AddCostRecord
	GetUserTotalCost = costs.GetUserTotalCost
	GetUserCostToday = costs.GetUserCostToday
	GetAllCostStats  = costs.GetAllCostStats
)
