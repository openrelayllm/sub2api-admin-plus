package proxy

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

func UseSecretCipher(encryptor service.SecretEncryptor) SecretCipher {
	return encryptor
}

func ProvideRuntimeConfig(cfg *config.Config) RuntimeConfig {
	return RuntimeConfigFromConfig(cfg)
}

func ProvideRuntime(cfg RuntimeConfig) Runtime {
	return NewLocalMihomoRuntime(cfg)
}

func ProvideNodeCheckCache(rdb *redis.Client) NodeCheckCache {
	return NewRedisNodeCheckCache(rdb)
}

func ProvideService(repo Repository, cipher SecretCipher, normalizer *SubscriptionNormalizer, runtime Runtime, runtimeCfg RuntimeConfig, cache NodeCheckCache) *Service {
	return NewService(repo, cipher, normalizer, runtime, runtimeCfg).WithNodeCheckCache(cache)
}

var ProviderSet = wire.NewSet(
	NewSQLRepository,
	NewSubscriptionNormalizer,
	UseSecretCipher,
	ProvideRuntimeConfig,
	ProvideRuntime,
	ProvideNodeCheckCache,
	ProvideService,
	wire.Bind(new(Repository), new(*SQLRepository)),
)
