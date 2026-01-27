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
	query := "SELECT k.id, k.userid, k.status, k.kobologin, k.kobolink, k.koboname, k.gslink, k.gsname, k.sheetname, g.ccode, k.lastresult FROM model_kobo_g_s k LEFT JOIN model_users_api_g_s g ON k.userid = g.userid WHERE k.status = 1"
	rows, err := r.db.Query(query)
	if err != nil {
		return results, err
	}
	defer rows.Close()
	for rows.Next() {
		result := models.Data{}
		if err = rows.Scan(
			&result.Id,
			&result.UserId,
			&result.Status,
			&result.KoboToken,
			&result.CSVLink,
			&result.FormName,
			&result.SpreadSheetID,
			&result.SpreadSheetName,
			&result.SheetName,
			&result.APIKey,
			&result.LastResult,
		); err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (r *Requests) WriteInfo(id int, info string) error {
	if len(info) > 254 {
		info = string([]rune(info)[:254])
	}
	query := "UPDATE model_kobo_g_s SET lastresult = ? WHERE id = ?"
	_, err := r.db.Exec(query, info, id)
	return err
}
