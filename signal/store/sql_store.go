package store

import (
	"context"

	"gorm.io/gorm"

	ypeer "yiji.one/punch/signal/store/peer"
)

type SqlStore struct {
	db *gorm.DB
}

func NewSqlStore(ctx context.Context, db *gorm.DB) *SqlStore {

	return &SqlStore{db: db}
}

func (s *SqlStore) GetPeers(token string) *[]ypeer.Peer {
	peers := &[]ypeer.Peer{}
	DB.Where("token = ?", token).Find(&ypeer.Peer{})
	return peers
}
