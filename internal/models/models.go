package models

import "database/sql"

type Data struct {
	Id              int
	UserId          int
	Status          int
	KoboToken       string
	CSVLink         string
	FormName        string
	SpreadSheetID   string
	SpreadSheetName string
	SheetName       string
	APIKey          string
	LastResult      sql.NullString
}
