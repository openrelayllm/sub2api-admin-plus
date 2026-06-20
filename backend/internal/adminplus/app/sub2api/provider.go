package sub2api

import (
	"database/sql"
	"os"
	"time"

	"github.com/google/wire"
	_ "github.com/lib/pq"
)

var ProviderSet = wire.NewSet(
	ProvideReadSQLDB,
	NewSQLRepository,
	wire.Bind(new(Repository), new(*SQLRepository)),
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
