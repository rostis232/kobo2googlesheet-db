package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/rostis232/kobo2googlesheet-db/internal/app/logwriter"
	"github.com/tealeg/xlsx/v3"
	"io"
	"net/http"
	"strings"
)

func (e *ExpImp) ExportXLS(xlsLink string, token string, client *http.Client) (map[string][][]string, error) {
	var allRecords = make(map[string][][]string)

	cutedLink, founded := strings.CutPrefix(xlsLink, "https://kobo.humanitarianresponse.info/")
	if founded {
		xlsLink = "https://eu.kobotoolbox.org/" + cutedLink
		logwriter.WriteLogToFile(fmt.Sprintf("Founded old URL, changed to new domen: %s", xlsLink))
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
		return allRecords, errors.New(response.Status)
	}

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	workbook, err := xlsx.OpenReaderAt(bytes.NewReader(buf.Bytes()), bytes.NewReader(buf.Bytes()).Size())
	if err != nil {
		return allRecords, err
	}

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
