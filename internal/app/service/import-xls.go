package service

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func (e *ExpImp) ImporterXLS(credentials string, spreadsheetId string, records map[string][][]string) error {
	var err error

	values := e.StringMapToInterfaceMapConverter(records)

	ctx := context.Background()

	credBytes, err := b64.StdEncoding.DecodeString(credentials)
	if err != nil {
		return err
	}

	config, err := google.JWTConfigFromJSON(credBytes, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return err
	}

	client := config.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}

	for sheetName, sheetData := range values {
		// Перевіряємо, чи існує аркуш
		exists := false
		spreadSheet, err := srv.Spreadsheets.Get(spreadsheetId).Context(ctx).Do()
		if err != nil {
			return err
		}

		for _, s := range spreadSheet.Sheets {
			if s.Properties.Title == sheetName {
				exists = true
				break
			}
		}

		// Якщо аркуш не існує, створюємо новий
		if !exists {
			_, err = srv.Spreadsheets.BatchUpdate(spreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{
				Requests: []*sheets.Request{
					{
						AddSheet: &sheets.AddSheetRequest{
							Properties: &sheets.SheetProperties{
								Title: sheetName,
							},
						},
					},
				},
			}).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to add new spreadSheet: %s", err)
			}
		}

		// Оновлюємо значення у визначеному діапазоні
		row := &sheets.ValueRange{
			Values: sheetData,
		}

		_, err = srv.Spreadsheets.Values.Update(spreadsheetId, sheetName, row).ValueInputOption("USER_ENTERED").Context(ctx).Do()
		if err != nil {
			return err
		}
	}

	return nil
}
