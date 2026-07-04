package actions

import "github.com/google/wire"

func ProvideService(repo Repository, supplierUpdater SupplierStatusUpdater) *Service {
	return NewServiceWithDependencies(repo, supplierUpdater)
}

var ProviderSet = wire.NewSet(
	NewSQLRepository,
	wire.Bind(new(Repository), new(*SQLRepository)),
	ProvideService,
)
