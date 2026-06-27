package purity

import (
	coreservice "github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

func ProvideService(repo Repository, accountResolver coreservice.AccountRepository) *Service {
	return NewServiceWithAccountResolver(repo, accountResolver)
}

var ProviderSet = wire.NewSet(
	NewSQLRepository,
	wire.Bind(new(Repository), new(*SQLRepository)),
	ProvideService,
)
