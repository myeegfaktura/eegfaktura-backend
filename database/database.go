package database

import (
	"database/sql"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

type OpenDbXConnection func() (*sqlx.DB, error)

// applyPoolSettings applies optional pool tuning from viper to the
// given DB handle. Missing keys leave the Go default in place (which
// is unlimited MaxOpenConns + MaxIdleConns=2 + no MaxLifetime — fine
// for tests, risky for production load).
//
// Recognised keys (all under `database.`):
//   maxOpenConns      int       — cap on open connections
//   maxIdleConns      int       — cap on idle connections
//   connMaxLifetime   duration  — e.g. "5m"; rotated after this age
func applyPoolSettings(db *sql.DB) {
	if max := viper.GetInt("database.maxOpenConns"); max > 0 {
		db.SetMaxOpenConns(max)
	}
	if max := viper.GetInt("database.maxIdleConns"); max > 0 {
		db.SetMaxIdleConns(max)
	}
	if lifetime := viper.GetDuration("database.connMaxLifetime"); lifetime > 0 {
		db.SetConnMaxLifetime(lifetime)
	}
}

func GetDBConnection() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		viper.GetString("database.host"), viper.GetInt("database.port"), viper.GetString("database.user"),
		viper.GetString("database.password"), viper.GetString("database.dbname"))
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	applyPoolSettings(db)
	return db, nil
}

func GetDBXConnection() (*sqlx.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		viper.GetString("database.host"), viper.GetInt("database.port"), viper.GetString("database.user"),
		viper.GetString("database.password"), viper.GetString("database.dbname"))
	db, err := sqlx.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}
	applyPoolSettings(db.DB)
	return db, nil
}

var pgDialect = goqu.Dialect("postgres")
