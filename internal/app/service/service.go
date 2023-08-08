package service

import "github.com/rostis232/kobo2googlesheet-db/internal/app/repository"

type ExportImport interface {
	Export(csvLink, token string) ([][]string, error)
	Converter(strs [][]string) [][]interface{}
	Importer(credentials string, spreadsheetId string, sheetName string, values [][]interface{}) error
}

type Service struct {
	ExportImport
}

func NewService(repo repository.Repository) *Service {
	return &Service{
		ExportImport: NewExpImp(repo),
	}
}
