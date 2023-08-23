package repository

import (
	"database/sql"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
)

type Database interface {
	GetAllData() ([]models.Data, error)
	WriteInfo(id int, info string) error
}

type Repository struct {
	Database
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		Database: NewRequests(db),
	}
}
