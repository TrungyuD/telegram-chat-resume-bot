package store

import (
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/costs"
)

type CostRecord = costs.CostRecord

var (
	AddCostRecord    = costs.AddCostRecord
	GetUserTotalCost = costs.GetUserTotalCost
	GetUserCostToday = costs.GetUserCostToday
	GetAllCostStats  = costs.GetAllCostStats
)
