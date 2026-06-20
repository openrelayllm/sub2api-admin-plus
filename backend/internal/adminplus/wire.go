package adminplus

import (
	actionsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/actions"
	balancesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/balances"
	billingapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/billing"
	extensionapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/extension"
	healthapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/health"
	promotionsapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/promotions"
	ratesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/rates"
	reconciliationapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/reconciliation"
	sub2apiapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/sub2api"
	suppliersapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/suppliers"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	actionsapp.ProviderSet,
	balancesapp.ProviderSet,
	billingapp.ProviderSet,
	extensionapp.ProviderSet,
	healthapp.ProviderSet,
	promotionsapp.ProviderSet,
	ratesapp.ProviderSet,
	reconciliationapp.ProviderSet,
	sub2apiapp.ProviderSet,
	suppliersapp.ProviderSet,
)
