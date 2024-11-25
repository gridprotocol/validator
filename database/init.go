package database

import (
	// "database/sql"
	"grid-prover/logs"
	"os"
	"path/filepath"
	"time"

	// "gorm.io/driver/mysql"

	// _ "github.com/go-sql-driver/mysql"
	"github.com/mitchellh/go-homedir"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var GlobalDataBase *gorm.DB
var logger = logs.Logger("database")

func InitDatabase(path string) error {
	dir, err := homedir.Expand(path)
	if err != nil {
		return err
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0666)
		if err != nil {
			return err
		}
	}

	// dsn := "root@tcp(127.0.0.1:3306)/grid?charset=utf8mb4&parseTime=True&loc=Local"
	// mysqlDB, err := sql.Open("mysql", dsn)
	// if err != nil {
	// 	return err
	// }

	db, err := gorm.Open(sqlite.Open(filepath.Join(dir, "grid.db")), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// 设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(10)
	// 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)
	// 设置超时时间
	sqlDB.SetConnMaxLifetime(time.Second * 30)

	err = sqlDB.Ping()
	if err != nil {
		return err
	}
	db.AutoMigrate(&Order{}, &ProfitStore{}, &BlockNumber{}, &Provider{}, &NodeStore{})
	GlobalDataBase = db

	logger.Info("init database success")
	return nil
}

func RemoveDataBase(path string) error {
	dir, err := homedir.Expand(path)
	if err != nil {
		return err
	}

	databasePath := filepath.Join(dir, "grid.db")
	if _, err := os.Stat(databasePath); os.IsExist(err) {
		if err := os.Remove(databasePath); err != nil {
			return err
		}
	}

	return nil
}
