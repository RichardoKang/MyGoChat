package test

import (
	db2 "MyGoChat/pkg/db"
	"testing"

	_ "github.com/lib/pq"
)

func TestDBConnection(t *testing.T) {
	db := db2.GetDB()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get DB from gorm.DB: %v", err)
	}

	err = sqlDB.Ping() // Ping
	if err != nil {
		t.Fatalf("Failed to ping DB: %v", err)
	}
}
