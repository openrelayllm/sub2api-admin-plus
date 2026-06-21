package sessions

import (
	"github.com/Wei-Shaw/sub2api/internal/adminplus/ports"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

func UseCipher(encryptor service.SecretEncryptor) Cipher {
	return encryptor
}

func ProvideService(repo Repository, cipher Cipher, suppliers SupplierLookup, prober ports.SessionProbeAdapter, login ports.SessionLoginAdapter) *Service {
	return NewServiceWithDependencies(repo, cipher, suppliers, prober, login)
}

var ProviderSet = wire.NewSet(
	UseCipher,
	NewSQLRepository,
	wire.Bind(new(Repository), new(*SQLRepository)),
	ProvideService,
)
