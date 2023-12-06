package service

import (
	"context"
	b64 "encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/rostis232/kobo2googlesheet-db/internal/app/logwriter"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/repository"
	"github.com/rostis232/kobo2googlesheet-db/internal/models"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type ExpImp struct {
	repo repository.Database
}

func NewExpImp(repo repository.Database) *ExpImp {
	return &ExpImp{
		repo: repo,
	}
}

func (e *ExpImp) Export(csvLink string, token string, client *http.Client) ([][]string, error) {
	var allRecords [][]string

	cutedLink, founded := strings.CutPrefix(csvLink, "https://kobo.humanitarianresponse.info/")

	if founded {
		csvLink = "https://eu.kobotoolbox.org/" + cutedLink
		logwriter.WriteLogToFile(fmt.Sprintf("Founded old URL, changed to new domen: %s", csvLink))
	}

	request, err := http.NewRequest("GET", csvLink, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", "Token "+token)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	r := csv.NewReader(strings.NewReader(string(body)))
	r.Comma = ';'
	r.Comment = '#'


	for {
        // Зчитування одного рядка CSV
        record, err := r.Read()
        if err == io.EOF {
            break // Вийти з циклу, якщо файл закінчився
        } else if err != nil {
            return nil, err // Повернути помилку, якщо сталася інша помилка
        }

        // Додати рядок до загального срізу
        allRecords = append(allRecords, record)
    }

	
	// records, err := r.ReadAll()
	// if err != nil {
	// 	return nil, err
	// }

	return allRecords, nil
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

func (e *ExpImp) Importer(credentials string, spreadSheetName string, spreadsheetId string, sheetName string, records [][]string) error {
	var err error
	var decr int = 1

	if !strings.Contains(sheetName, "!") {
		sheetName += "!A1:XYZ"
	}

	numberOfRows := getStringNumber(sheetName)

	if strings.Contains(spreadSheetName, " -wot") {
		decr = 2
	}


	if strings.Contains(spreadSheetName, " -idx") {
		fmt.Printf("%s: Founded -idx: changing index\n", spreadSheetName)
		records, err = changingIndex(records, numberOfRows, decr)
		if err != nil {
			return fmt.Errorf("error while changing indexes: %s", err)
		}
	}

	if strings.Contains(spreadSheetName, " -wot") {
		records = records[1:]
		fmt.Printf("%s: Founded -wot: deleted titles\n", spreadSheetName)
	}

	values := e.Converter(records)


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

	row := &sheets.ValueRange{
		Values: values,
	}


	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, sheetName, row).ValueInputOption("USER_ENTERED").Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}

// Sorter sorts data at first by API Key and at second - by CSV Link
func (e *ExpImp) Sorter(data []models.Data) map[string]map[string][]models.Data {
	dataByAPIKey := make(map[string][]models.Data)

	for _, d := range data {
		dataByAPIKey[d.APIKey] = append(dataByAPIKey[d.APIKey], d)
	}

	dataByAPIKeyAndCSVlink := make(map[string]map[string][]models.Data)

	for k, v := range dataByAPIKey {
		byCSV := make(map[string][]models.Data)

		for _, n := range v {
			byCSV[n.CSVLink] = append(byCSV[n.CSVLink], n)
		}

		dataByAPIKeyAndCSVlink[k] = byCSV
	}

	return dataByAPIKeyAndCSVlink
}

func changingIndex (input [][]string, numberOfRows int, decr int) ([][]string, error) {
	inputCopy := make([][]string, len(input))

	for i := range input {
        inputCopy[i] = make([]string, len(input[i]))
        copy(inputCopy[i], input[i])
    }

	indexId := 0
	for rowId, cells := range inputCopy {
		for cellId, cellValue := range cells {
			if rowId == 0 {
				if cellValue == "_index" {
					indexId = cellId
				}
			} else {
				if indexId == 0 {
					return inputCopy, fmt.Errorf("index column not found")
				} else {
					if cellId == indexId {
						indexValueInd, err := strconv.Atoi(cellValue)
						if err != nil {
							return inputCopy, fmt.Errorf("error while converting string to ind")
						}
						strValue := strconv.Itoa(numberOfRows + indexValueInd - decr)

						inputCopy[rowId][cellId] = strValue
					}
				}
			}
		}
	}
	return inputCopy, nil
}


func getStringNumber(sheetRange string) int {
	_, after, ok:= strings.Cut(sheetRange, "!A")
	if !ok {
		fmt.Println("Error while getting string number (poin 1)")
	}
	before, _, ok := strings.Cut(after, ":")
	if !ok {
		fmt.Println("Error while getting string number (poin 2)")
	}
	number, err := strconv.Atoi(before)
	if err != nil {
		fmt.Println("Error while getting string number (poin 2)")
	}
	return number
}