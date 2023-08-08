package repository

import (
	"database/sql"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
)

type Requests struct {
	db *sql.DB
}

func NewRequests(db *sql.DB) *Requests {
	return &Requests{
		db: db,
	}
}

func (r *Requests) GetAllData() ([]models.Data, error) {
	results := []models.Data{}
	query := "SELECT k.userid, k.kobologin, k.kobolink, k.koboname, k.gslink, k.gsname, k.sheetname, g.ccode FROM model_kobo_g_s k LEFT JOIN model_users_api_g_s g ON k.userid = g.userid"
	rows, err := r.db.Query(query)
	if err != nil {
		return results, err
	}
	defer rows.Close()
	for rows.Next() {
		result := models.Data{}
		if err = rows.Scan(
			&result.UserId,
			&result.KoboToken,
			&result.CSVLink,
			&result.FormName,
			&result.SpreadSheetID,
			&result.SpreadSheetName,
			&result.SheetName,
			&result.APIKey,
		); err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}
