package service

import (
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
	"net/http"
)

type ExportImport interface {
	Export(csvLink, token string, client *http.Client) ([][]string, error)
	Converter(strs [][]string) [][]interface{}
	Importer(credentials string, spreadSheetName string, spreadsheetId string, sheetName string, values [][]string) error
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
