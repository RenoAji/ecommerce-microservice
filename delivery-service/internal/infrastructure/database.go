package infrastructure

import (
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitializeDatabase(dsnString string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	for i := 0; i < 10; i++ {	
		db, err = gorm.Open(postgres.Open(dsnString), &gorm.Config{})
		if err == nil {
			return db, nil
		}
		log.Printf("Waiting for database... attempt %d", i+1)
		time.Sleep(2 * time.Second)
	}

	return nil, err
}