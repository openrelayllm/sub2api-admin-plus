package sub2api

import (
	"database/sql"
	"os"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/wire"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var ProviderSet = wire.NewSet(
	ProvideReadSQLDB,
	ProvideReadRedis,
	NewSQLRepository,
	NewRuntimeRepository,
	wire.Bind(new(Repository), new(*SQLRepository)),
	wire.Bind(new(RuntimeReader), new(*RuntimeRepository)),
	NewService,
)

type ReadDB struct {
	DB *sql.DB
}

func ProvideReadSQLDB(defaultDB *sql.DB) ReadDB {
	dsn := os.Getenv("SUB2API_READONLY_DATABASE_URL")
	if dsn == "" {
		return ReadDB{DB: defaultDB}
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return ReadDB{DB: defaultDB}
	}
	db.SetMaxOpenConns(16)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(30 * time.Minute)
	return ReadDB{DB: db}
}

type Sub2APIRedis struct {
	Client     *redis.Client
	Configured bool
	Owned      bool
}

func ProvideReadRedis(defaultRedis *redis.Client, cfg *config.Config) Sub2APIRedis {
	redisURL := os.Getenv("SUB2API_READONLY_REDIS_URL")
	dbOverride := os.Getenv("SUB2API_READONLY_REDIS_DB")
	if redisURL == "" && dbOverride == "" {
		return Sub2APIRedis{Client: defaultRedis, Configured: defaultRedis != nil}
	}

	var opts *redis.Options
	if redisURL != "" {
		parsed, err := redis.ParseURL(redisURL)
		if err != nil {
			return Sub2APIRedis{Client: defaultRedis, Configured: defaultRedis != nil}
		}
		opts = parsed
	} else {
		opts = &redis.Options{
			Addr:         cfg.Redis.Address(),
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			DialTimeout:  time.Duration(cfg.Redis.DialTimeoutSeconds) * time.Second,
			ReadTimeout:  time.Duration(cfg.Redis.ReadTimeoutSeconds) * time.Second,
			WriteTimeout: time.Duration(cfg.Redis.WriteTimeoutSeconds) * time.Second,
			PoolSize:     cfg.Redis.PoolSize,
			MinIdleConns: cfg.Redis.MinIdleConns,
		}
	}
	if dbOverride != "" {
		db, err := strconv.Atoi(dbOverride)
		if err != nil || db < 0 {
			return Sub2APIRedis{Client: defaultRedis, Configured: defaultRedis != nil}
		}
		opts.DB = db
	}
	return Sub2APIRedis{Client: redis.NewClient(opts), Configured: true, Owned: true}
}
