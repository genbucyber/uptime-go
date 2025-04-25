package database

import (
	"fmt"
	"os"
	"sync"
	"time"

	"uptime-go/internal/net/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const (
	DBPath = "/var/uptime-go/db/uptime.db"
)

type Database struct {
	DB    *gorm.DB
	mutex sync.RWMutex
}

func InitializeDatabase() (*gorm.DB, error) {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll("/var/uptime-go/db", 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if the database file exists, if not create it
	if _, err := os.Stat(DBPath); os.IsNotExist(err) {
		file, err := os.Create(DBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create database file: %w", err)
		}
		defer file.Close()
	}

	// Open the database connection using GORM and SQLite with connection pool configuration
	db, err := gorm.Open(sqlite.Open(DBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pooling
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Set connection pool settings
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns sets the maximum number of open connections to the database
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused
	sqlDB.SetConnMaxLifetime(time.Hour)

	// SetConnMaxIdleTime sets the maximum amount of time a connection may be idle
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	// Migrate the schema
	if err := db.AutoMigrate(&config.CheckResults{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database schema: %w", err)
	}

	return db, nil
}

func (db *Database) SaveResults(results *config.CheckResults) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(results).Error; err != nil {
			return fmt.Errorf("failed to save results: %w", err)
		}
		return nil
	})
	return err
}
