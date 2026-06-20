package adminplus

import (
	ratesapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/rates"
	suppliersapp "github.com/Wei-Shaw/sub2api/internal/adminplus/app/suppliers"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	ratesapp.ProviderSet,
	suppliersapp.ProviderSet,
)
