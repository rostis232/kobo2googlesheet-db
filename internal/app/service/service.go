package service

import (
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
)

type ExportImport interface {
	Export(csvLink, token string) ([][]string, error)
	Converter(strs [][]string) [][]interface{}
	Importer(credentials string, spreadsheetId string, sheetName string, values [][]interface{}) error
	Sorter(data []models.Data) map[string]map[string][]models.Data
}

type Service struct {
	ExportImport
}

func NewService(repo repository.Repository) *Service {
	return &Service{
		ExportImport: NewExpImp(repo),
	}
}
