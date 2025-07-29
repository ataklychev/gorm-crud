package gorm_crud

import (
	"gorm.io/gorm"
)

type BaseRepositoryInterface interface {
}

type BaseRepository struct {
	BaseRepositoryInterface
	Db     *gorm.DB
	Logger LoggerInterface
}

func NewBaseRepository(db *gorm.DB, logger LoggerInterface) *BaseRepository {
	return &BaseRepository{Db: db, Logger: logger}
}
