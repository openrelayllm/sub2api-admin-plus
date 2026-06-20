package suppliers

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
)

func UseCredentialCipher(encryptor service.SecretEncryptor) CredentialCipher {
	return encryptor
}

var ProviderSet = wire.NewSet(
	UseCredentialCipher,
	NewSQLRepositoryWithCipher,
	wire.Bind(new(Repository), new(*SQLRepository)),
	NewService,
)
