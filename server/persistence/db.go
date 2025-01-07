package persistence

import (
	"sync"

	"Goauld/common/log"
	"Goauld/server/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type DB struct {
	db *gorm.DB
}

var dbTypes = []any{
	Agent{},
}

var (
	once sync.Once
	db   *DB
)

// InitDB initialize the database, performing migrations if needed
func InitDB() (*DB, error) {
	db := get()
	err := db.Migrate()
	if err != nil {
		return nil, err
	}
	return db, nil
}

// get return the database connection object
func get() *DB {
	var err error
	once.Do(func() {
		dbFileName := ""
		if config.Get().NoDB {
			dbFileName = ":memory:"
		} else {
			dbFileName = config.Get().DbFileName
		}

		_db, _err := gorm.Open(sqlite.Open(dbFileName), &gorm.Config{
			Logger: log.GetGormLogger(),
		})
		if err != nil {
			err = _err
			return
		}
		db = &DB{db: _db}
	})
	return db
}

// Migrate performs the database migration
func (db *DB) Migrate() error {
	for _, dbType := range dbTypes {
		err := db.db.AutoMigrate(&dbType)
		if err != nil {
			return err
		}
	}
	return nil
}
