package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

type DB struct {
	db *gorm.DB
}

var (
	dbTypes = []any{
		Agent{},
	}
)

var once sync.Once
var _db *DB

func InitDB() (*DB, error) {
	db := Get()
	err := db.Migrate()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func Get() *DB {
	var err error
	once.Do(func() {
		__db, _err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
		if err != nil {
			err = _err
			return
		}
		_db = &DB{db: __db}
	})

	return _db
}

func (db *DB) Migrate() error {
	for _, dbType := range dbTypes {
		err := db.db.AutoMigrate(&dbType)
		if err != nil {
			return err
		}
	}
	return nil
}
