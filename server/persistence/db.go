package persistence

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
)

var once sync.Once
var db *gorm.DB

func Get() *gorm.DB {
	var err error
	once.Do(func() {
		db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
		if err != nil {
			return
		}
	})
	return db
}
