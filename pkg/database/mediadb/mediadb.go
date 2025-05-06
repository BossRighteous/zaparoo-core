package mediadb

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZaparooProject/zaparoo-core/pkg/database"
	"github.com/ZaparooProject/zaparoo-core/pkg/database/systemdefs"
	"github.com/ZaparooProject/zaparoo-core/pkg/utils"
	_ "modernc.org/sqlite"
)

var ErrorNullSql = errors.New("MediaDB is not connected")

type MediaDB struct {
	sql    *sql.DB
	dbPath string
}

func OpenMediaDB(dbPath string) (*MediaDB, error) {
	db := &MediaDB{sql: nil, dbPath: dbPath}
	err := db.Open()
	return db, err
}

// Returns copied disk db as in-memory MediaDB instance
// Method for interface ease
func (mdb *MediaDB) OpenTempMediaDB() (database.MediaDBI, error) {
	tdb := &MediaDB{sql: nil, dbPath: mdb.dbPath}
	memPath := "file::memory:?cache=shared"
	tsql, err := sql.Open("sqlite", memPath)
	if err != nil {
		return tdb, err
	}
	tdb.sql = tsql
	tsql.Ping()
	msql := mdb.UnsafeGetSqlDB()
	_, err = msql.Exec(`vacuum into ?`, memPath)
	msql.Exec(`update Media set IsActive = 0`)
	return tdb, err
}

// Closes tempDB instance after vacuuming into original mediaDB path
// PLEASE close AND reopen MediaDB around this to be safe?
func (tdb *MediaDB) CloseTempMediaDB() error {
	sql := tdb.UnsafeGetSqlDB()
	_, err := sql.Exec(`vacuum into ?`, tdb.dbPath)
	tdb.Close()
	return err
}

func (db *MediaDB) Open() error {
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

func (db *MediaDB) GetDBPath() string {
	return db.dbPath
}

func (db *MediaDB) Exists() bool {
	return db.sql != nil
}

func (db *MediaDB) UnsafeGetSqlDB() *sql.DB {
	return db.sql
}

func (db *MediaDB) Truncate() error {
	if db.sql == nil {
		return ErrorNullSql
	}
	return sqlTruncate(db.sql)
}

func (db *MediaDB) Allocate() error {
	if db.sql == nil {
		return ErrorNullSql
	}
	return sqlAllocate(db.sql)
}

func (db *MediaDB) Vacuum() error {
	if db.sql == nil {
		return ErrorNullSql
	}
	return sqlVacuum(db.sql)
}

func (db *MediaDB) Close() error {
	if db.sql == nil {
		return nil
	}
	return db.sql.Close()
}

func (db *MediaDB) BeginTransaction() error {
	return sqlBeginTransaction(db.sql)
}

func (db *MediaDB) CommitTransaction() error {
	return sqlCommitTransaction(db.sql)
}

func (db *MediaDB) CleanInactiveMedia() error {
	return sqlCleanInactiveMedia(db.sql)
}

// Return indexed names matching exact query (case insensitive).
func (db *MediaDB) SearchMediaPathExact(systems []systemdefs.System, query string) ([]database.SearchResult, error) {
	if db.sql == nil {
		return make([]database.SearchResult, 0), ErrorNullSql
	}
	return sqlSearchMediaPathExact(db.sql, systems, query)
}

// Return indexed names that include every word in query (case insensitive).
func (db *MediaDB) SearchMediaPathWords(systems []systemdefs.System, query string) ([]database.SearchResult, error) {
	if db.sql == nil {
		return make([]database.SearchResult, 0), ErrorNullSql
	}
	qWords := strings.Fields(strings.ToLower(query))
	return sqlSearchMediaPathParts(db.sql, systems, qWords)
}

// Glob pattern matching unclear on some patterns
func (db *MediaDB) SearchMediaPathGlob(systems []systemdefs.System, query string) ([]database.SearchResult, error) {
	// query == path like with possible *
	var nullResults []database.SearchResult
	if db.sql == nil {
		return nullResults, ErrorNullSql
	}
	var parts []string
	for _, part := range strings.Split(query, "*") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		// return random instead
		rnd, err := db.RandomGame(systems)
		if err != nil {
			return nullResults, err
		}
		return []database.SearchResult{rnd}, nil
	}

	return sqlSearchMediaPathParts(db.sql, systems, parts)
	// TODO since we approximated a glob, we should actually check
	// result paths against base glob to confirm
}

// Return true if a specific system is indexed in the gamesdb
func (db *MediaDB) SystemIndexed(system systemdefs.System) bool {
	if db.sql == nil {
		return false
	}
	return sqlSystemIndexed(db.sql, system)
}

// Return all systems indexed in the gamesdb
func (db *MediaDB) IndexedSystems() ([]string, error) {
	// JBONE: return string map of Systems.Key, Systems.Indexed
	var systems []string
	if db.sql == nil {
		return systems, ErrorNullSql
	}
	return sqlIndexedSystems(db.sql)
}

// Return a random game from specified systems.
func (db *MediaDB) RandomGame(systems []systemdefs.System) (database.SearchResult, error) {
	var result database.SearchResult
	if db.sql == nil {
		return result, ErrorNullSql
	}

	system, err := utils.RandomElem(systems)
	if err != nil {
		return result, err
	}

	return sqlRandomGame(db.sql, system)
}

func (db *MediaDB) FindSystem(row database.System) (database.System, error) {
	return sqlFindSystem(db.sql, row)
}

func (db *MediaDB) InsertSystem(row database.System) (database.System, error) {
	return sqlInsertSystem(db.sql, row)
}

func (db *MediaDB) FindOrInsertSystem(row database.System) (database.System, error) {
	system, err := db.FindSystem(row)
	if err == sql.ErrNoRows {
		system, err = db.InsertSystem(row)
	}
	return system, err
}

func (db *MediaDB) FindMediaTitle(row database.MediaTitle) (database.MediaTitle, error) {
	return sqlFindMediaTitle(db.sql, row)
}

func (db *MediaDB) InsertMediaTitle(row database.MediaTitle) (database.MediaTitle, error) {
	return sqlInsertMediaTitle(db.sql, row)
}

func (db *MediaDB) FindOrInsertMediaTitle(row database.MediaTitle) (database.MediaTitle, error) {
	system, err := db.FindMediaTitle(row)
	if err == sql.ErrNoRows {
		system, err = db.InsertMediaTitle(row)
	}
	return system, err
}

func (db *MediaDB) FindMedia(row database.Media) (database.Media, error) {
	return sqlFindMedia(db.sql, row)
}

func (db *MediaDB) InsertMedia(row database.Media) (database.Media, error) {
	return sqlInsertMedia(db.sql, row)
}

func (db *MediaDB) FindOrInsertMedia(row database.Media) (database.Media, error) {
	system, err := db.FindMedia(row)
	if err == sql.ErrNoRows {
		system, err = db.InsertMedia(row)
	}
	return system, err
}

func (db *MediaDB) FindTagType(row database.TagType) (database.TagType, error) {
	return sqlFindTagType(db.sql, row)
}

func (db *MediaDB) InsertTagType(row database.TagType) (database.TagType, error) {
	return sqlInsertTagType(db.sql, row)
}

func (db *MediaDB) FindOrInsertTagType(row database.TagType) (database.TagType, error) {
	system, err := db.FindTagType(row)
	if err == sql.ErrNoRows {
		system, err = db.InsertTagType(row)
	}
	return system, err
}

func (db *MediaDB) FindTag(row database.Tag) (database.Tag, error) {
	return sqlFindTag(db.sql, row)
}

func (db *MediaDB) InsertTag(row database.Tag) (database.Tag, error) {
	return sqlInsertTag(db.sql, row)
}

func (db *MediaDB) FindOrInsertTag(row database.Tag) (database.Tag, error) {
	system, err := db.FindTag(row)
	if err == sql.ErrNoRows {
		system, err = db.InsertTag(row)
	}
	return system, err
}

func (db *MediaDB) FindMediaTag(row database.MediaTag) (database.MediaTag, error) {
	return sqlFindMediaTag(db.sql, row)
}

func (db *MediaDB) InsertMediaTag(row database.MediaTag) (database.MediaTag, error) {
	return sqlInsertMediaTag(db.sql, row)
}

func (db *MediaDB) FindOrInsertMediaTag(row database.MediaTag) (database.MediaTag, error) {
	system, err := db.FindMediaTag(row)
	if err == sql.ErrNoRows {
		system, err = db.InsertMediaTag(row)
	}
	return system, err
}
