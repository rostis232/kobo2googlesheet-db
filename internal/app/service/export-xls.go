package service

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/tealeg/xlsx/v3"
	"io"
	"net/http"
	"os"
	"strings"
)

func (e *ExpImp) ExportXLS(xlsLink string, token string, client *http.Client) (map[string][][]string, error) {
	var allRecords = make(map[string][][]string)

	cutedLink, founded := strings.CutPrefix(xlsLink, "https://kobo.humanitarianresponse.info/")
	if founded {
		xlsLink = "https://eu.kobotoolbox.org/" + cutedLink
		logrus.WithFields(logrus.Fields{"new_url": xlsLink}).Info("Founded old URL, changed to new domain")
	}

	request, err := http.NewRequest("GET", xlsLink, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Add("Authorization", "Token "+token)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d %s", response.StatusCode, response.Status)
	}

	tempFile, err := os.CreateTemp("", "kobo-*.xlsx")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, response.Body); err != nil {
		return nil, fmt.Errorf("failed to save response to temp file: %w", err)
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", err)
	}

	workbook, err := xlsx.OpenFile(tempFile.Name())
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, sheet := range workbook.Sheets {
			sheet.Close()
		}
	}()

	sheets := workbook.Sheets

	for _, sheet := range sheets {
		sheetRecords := [][]string{}

		err = sheet.ForEachRow(func(row *xlsx.Row) error {
			rowRecords := []string{}

			err = row.ForEachCell(func(cell *xlsx.Cell) error {
				rowRecords = append(rowRecords, cell.String())
				return nil
			})
			if err != nil {
				return err
			}

			sheetRecords = append(sheetRecords, rowRecords)
			return nil
		})

		if err != nil {
			return allRecords, err
		}

		allRecords[sheet.Name] = sheetRecords
	}

	return allRecords, nil
}
