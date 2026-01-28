package service

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xuri/excelize/v2"
	"io"
	"net/http"
	"os"
	"strings"
)

func (e *ExpImp) ExportXLS(xlsLink string, token string, client *http.Client, callback func(sheetName string, records [][]string) error) error {
	cutedLink, founded := strings.CutPrefix(xlsLink, "https://kobo.humanitarianresponse.info/")
	if founded {
		xlsLink = "https://eu.kobotoolbox.org/" + cutedLink
		logrus.WithFields(logrus.Fields{"new_url": xlsLink}).Info("Founded old URL, changed to new domain")
	}

	request, err := http.NewRequest("GET", xlsLink, nil)
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", "Token "+token)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d %s", response.StatusCode, response.Status)
	}

	tempFile, err := os.CreateTemp("", "kobo-*.xlsx")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, response.Body); err != nil {
		return fmt.Errorf("failed to save response to temp file: %w", err)
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek temp file: %w", err)
	}

	f, err := excelize.OpenFile(tempFile.Name())
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.WithError(err).Error("failed to close excelize file")
		}
	}()

	sheets := f.GetSheetList()

	for _, sheetName := range sheets {
		sheetRecords := [][]string{}

		rows, err := f.Rows(sheetName)
		if err != nil {
			return err
		}

		for rows.Next() {
			row, err := rows.Columns()
			if err != nil {
				rows.Close()
				return err
			}
			sheetRecords = append(sheetRecords, row)
		}

		if err = rows.Close(); err != nil {
			return err
		}

		if err := callback(sheetName, sheetRecords); err != nil {
			return err
		}
	}

	return nil
}
