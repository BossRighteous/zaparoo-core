package userdb

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"github.com/ZaparooProject/zaparoo-core/pkg/database"
	_ "modernc.org/sqlite"
)

var ErrorNullSql = errors.New("UserDB is not connected")

type UserDB struct {
	sql    *sql.DB
	dbPath string
}

func OpenUserDB(dbPath string) (*UserDB, error) {
	db := &UserDB{sql: nil, dbPath: dbPath}
	err := db.Open()
	return db, err
}

func (db *UserDB) Open() error {
	exists := true
	_, err := os.Stat(db.dbPath)
	if err != nil {
		exists = false
		os.MkdirAll(filepath.Dir(db.dbPath), 0755)
		//if err != nil {
		//	return err
		//}
	}
	sql, err := sql.Open("sqlite", db.dbPath)
	if err != nil {
		return err
	}
	db.sql = sql
	if !exists {
		return db.Allocate()
	}
	return nil
}

func (db *UserDB) GetDBPath() string {
	return db.dbPath
}

func (db *UserDB) UnsafeGetSqlDB() *sql.DB {
	return db.sql
}

func (db *UserDB) Truncate() error {
	if db.sql == nil {
		return ErrorNullSql
	}
	return sqlTruncate(db.sql)
}

func (db *UserDB) Allocate() error {
	if db.sql == nil {
		return ErrorNullSql
	}
	return sqlAllocate(db.sql)
}

func (db *UserDB) Vacuum() error {
	if db.sql == nil {
		return ErrorNullSql
	}
	return sqlVacuum(db.sql)
}

func (db *UserDB) Close() error {
	if db.sql == nil {
		return nil
	}
	return db.sql.Close()
}

// TODO: reader source (physical reader vs web)
// TODO: metadata

func (db *UserDB) AddHistory(entry database.HistoryEntry) error {
	return sqlAddHistory(db.sql, entry)
}

func (db *UserDB) GetHistory(lastId int) ([]database.HistoryEntry, error) {
	return sqlGetHistoryWithOffset(db.sql, lastId)
}
