package kanban

import "github.com/google/wire"

func ProvideService(repo Repository, siteCatalog SiteCatalogReader, evidenceScheduler AcceptanceEvidenceScheduler) *Service {
	return NewServiceWithAllDependencies(repo, siteCatalog, evidenceScheduler)
}

var ProviderSet = wire.NewSet(
	NewSQLRepository,
	wire.Bind(new(Repository), new(*SQLRepository)),
	ProvideService,
)
