package models

type Data struct {
	UserId          *int    `db:"userid"`
	KoboToken       *string `db:"kobologin"`
	CSVLink         *string `db:"kobolink"`
	FormName        *string `db:"koboname"`
	SpreadSheetID   *string `db:"gslink"`
	SpreadSheetName *string `db:"gsname"`
	SheetName       *string `db:"sheetname"`
	APIKey          *string `db:"ccode"`
}
