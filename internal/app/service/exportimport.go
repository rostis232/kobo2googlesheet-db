package service

import (
	"context"
	b64 "encoding/base64"
	"encoding/csv"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"io"
	"net/http"
	"strings"
)

type ExpImp struct {
	repo repository.Database
}

func NewExpImp(repo repository.Database) *ExpImp {
	return &ExpImp{
		repo: repo,
	}
}

func (e *ExpImp) Export(csvLink, token string) ([][]string, error) {
	client := &http.Client{}

	request, err := http.NewRequest("GET", csvLink, nil)
	if err != nil {
		return [][]string{{}}, err
	}

	request.Header.Add("Authorization", "Token "+token)

	response, err := client.Do(request)
	if err != nil {
		return [][]string{{}}, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return [][]string{{}}, err
	}

	defer response.Body.Close()

	r := csv.NewReader(strings.NewReader(string(body)))
	r.Comma = ';'
	r.Comment = '#'

	records, err := r.ReadAll()
	if err != nil {
		return [][]string{{}}, err
	}

	return records, nil
}

func (e *ExpImp) Converter(strs [][]string) [][]interface{} {
	var result [][]interface{}
	for _, row := range strs {
		var interfaceRow []interface{}
		for _, item := range row {
			newItem := item
			if strings.HasPrefix(item, "+") {
				newItem = "'" + item
			}

			interfaceRow = append(interfaceRow, newItem)
		}
		result = append(result, interfaceRow)
	}
	return result
}

func (e *ExpImp) Importer(credentials string, spreadsheetId string, sheetName string, values [][]interface{}) error {
	ctx := context.Background()

	credBytes, err := b64.StdEncoding.DecodeString(credentials)
	if err != nil {
		//log.Println(err)
		return err
	}

	config, err := google.JWTConfigFromJSON(credBytes, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		//log.Println(err)
		return err
	}

	client := config.Client(ctx)

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		//log.Println(err)
		return err
	}

	row := &sheets.ValueRange{
		Values: values,
	}

	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, sheetName+"!A1:XYZ", row).ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		//log.Println(err)
		return err
	}

	return nil
}
