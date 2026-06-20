package extension

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewSQLRepository,
	wire.Bind(new(Repository), new(*SQLRepository)),
	NewIngestProcessor,
	wire.Bind(new(ResultProcessor), new(*IngestProcessor)),
	NewServiceWithDependencies,
)
