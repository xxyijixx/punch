package store

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	ypeer "yiji.one/punch/signal/store/peer"
)

const (
	storSqliteFileName = "./signal.db"
	idQueryCondition   = "id = ?"
)

var DB *gorm.DB

func init() {
	var err error
	// databases init
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Info,
			Colorful:      true,
		},
	)
	DB, err = gorm.Open(sqlite.Open(storSqliteFileName), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	DB.AutoMigrate(&ypeer.Peer{})
	// 如果有数据清空
	DB.Where("1 = 1").Delete(&ypeer.Peer{})
}
