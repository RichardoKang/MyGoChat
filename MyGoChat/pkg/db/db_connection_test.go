package db

import (
	"testing"

	_ "github.com/lib/pq"
)

func TestDBConnection(t *testing.T) {
	db := GetDB()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get DB from gorm.DB: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		t.Fatalf("Failed to ping DB: %v", err)
	}
}
