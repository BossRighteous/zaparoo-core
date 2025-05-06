package database

import (
	"database/sql"
	"time"

	"github.com/ZaparooProject/zaparoo-core/pkg/database/systemdefs"
)

/*
 * In attempting to correct circular import deps these non-concrete
 * interfaces were moves to this generic package level.
 * Actual implementations found in userdb/mediadb
 */

/*
 * Portable interface for ENV bindings
 */
type Database struct {
	UserDB  UserDBI
	MediaDB MediaDBI
}

/*
 * Structs for SQL records
 */
type HistoryEntry struct {
	DBID    int64
	Time    time.Time `json:"time"`
	Type    string    `json:"type"`
	UID     string    `json:"uid"`
	Text    string    `json:"text"`
	Data    string    `json:"data"`
	Success bool      `json:"success"`
}

type Mapping struct {
	DBID     int64
	Id       string `json:"id"`
	Added    int64  `json:"added"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"`
	Match    string `json:"match"`
	Pattern  string `json:"pattern"`
	Override string `json:"override"`
}

type System struct {
	DBID     int64
	SystemID string
	Name     string
}

type MediaTitle struct {
	DBID       int64
	SystemDBID int64
	Slug       string
	Name       string
}

type Media struct {
	DBID           int64
	MediaTitleDBID int64
	Path           string
}

type TagType struct {
	DBID int64
	Type string
}

type Tag struct {
	DBID     int64
	TypeDBID int64
	Tag      string
}

type MediaTag struct {
	DBID      int64
	MediaDBID int64
	TagDBID   int64
}

type SearchResult struct {
	SystemID string
	Name     string
	Path     string
}

type FileInfo struct {
	SystemID string
	Path     string
	Name     string
}

/*
 * Interfaces for external deps
 */

type GenericDBI interface {
	Open() error
	UnsafeGetSqlDB() *sql.DB
	Truncate() error
	Allocate() error
	Vacuum() error
	Close() error
	GetDBPath() string
}

type UserDBI interface {
	GenericDBI
	AddHistory(entry HistoryEntry) error
	GetHistory(lastId int) ([]HistoryEntry, error)
	AddMapping(m Mapping) error
	GetMapping(id string) (Mapping, error)
	DeleteMapping(id string) error
	UpdateMapping(id string, m Mapping) error
	GetAllMappings() ([]Mapping, error)
	GetEnabledMappings() ([]Mapping, error)
}

type MediaDBI interface {
	GenericDBI
	OpenTempMediaDB() (MediaDBI, error)
	CloseTempMediaDB() error
	BeginTransaction() error
	CommitTransaction() error
	CleanInactiveMedia() error
	Exists() bool

	SearchMediaPathExact(systems []systemdefs.System, query string) ([]SearchResult, error)
	SearchMediaPathWords(systems []systemdefs.System, query string) ([]SearchResult, error)
	SearchMediaPathGlob(systems []systemdefs.System, query string) ([]SearchResult, error)
	IndexedSystems() ([]string, error)
	SystemIndexed(system systemdefs.System) bool
	RandomGame(systems []systemdefs.System) (SearchResult, error)

	FindSystem(row System) (System, error)
	InsertSystem(row System) (System, error)
	FindOrInsertSystem(row System) (System, error)

	FindMediaTitle(row MediaTitle) (MediaTitle, error)
	InsertMediaTitle(row MediaTitle) (MediaTitle, error)
	FindOrInsertMediaTitle(row MediaTitle) (MediaTitle, error)

	FindMedia(row Media) (Media, error)
	InsertMedia(row Media) (Media, error)
	FindOrInsertMedia(row Media) (Media, error)

	FindTagType(row TagType) (TagType, error)
	InsertTagType(row TagType) (TagType, error)
	FindOrInsertTagType(row TagType) (TagType, error)

	FindTag(row Tag) (Tag, error)
	InsertTag(row Tag) (Tag, error)
	FindOrInsertTag(row Tag) (Tag, error)

	FindMediaTag(row MediaTag) (MediaTag, error)
	InsertMediaTag(row MediaTag) (MediaTag, error)
	FindOrInsertMediaTag(row MediaTag) (MediaTag, error)
}
