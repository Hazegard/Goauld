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
var db *DB

func Get() *DB {
	var err error
	once.Do(func() {
		_db, _err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
		if err != nil {
			err = _err
			return
		}
		db = &DB{db: _db}
	})
	return db
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
