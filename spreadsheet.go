package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/sheets/v4"
)

// sheets wrapper
type Service struct {
	*sheets.Service
}

type Spreadsheet struct {
	*sheets.Spreadsheet
	Service *sheets.Service
}

type Sheet struct {
	*sheets.Sheet
	Spreadsheet *Spreadsheet
}

func getClient(credentialFilePath string) (*http.Client, error) {
	data, err := ioutil.ReadFile(credentialFilePath)
	if err != nil {
		return nil, err
	}
	conf, err := google.JWTConfigFromJSON(data, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return nil, err
	}

	return conf.Client(oauth2.NoContext), nil
}

func NewService(credentialFile string) (*Service, error) {
	client, err := getClient(credentialFile)
	if err != nil {
		return nil, err
	}

	service, err := sheets.New(client)
	if err != nil {
		return nil, err
	}
	return &Service{Service: service}, nil
}

func (s *Service) Spreadsheet(spreadsheetId string) (*Spreadsheet, error) {
	spreadsheet, err := s.Spreadsheets.Get(spreadsheetId).Do()
	if err != nil {
		return nil, err
	}
	return &Spreadsheet{Spreadsheet: spreadsheet, Service: s.Service}, nil
}

func (s *Spreadsheet) Reload() error {
	spreadsheet, err := s.Service.Spreadsheets.Get(s.SpreadsheetId).Do()
	if err != nil {
		return err
	}
	s.Spreadsheet = spreadsheet
	return nil
}

func (s *Spreadsheet) SheetByName(name string) (*Sheet, error) {
	for _, sheet := range s.Spreadsheet.Sheets {
		if sheet.Properties.Title == name {
			return &Sheet{Spreadsheet: s, Sheet: sheet}, nil
		}
	}
	return nil, errors.New(name + " not found.")
}

func (s *Spreadsheet) AddSheet(name string, cols, rows int64) (*Sheet, error) {
	reqs := []*sheets.Request{
		&sheets.Request{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Title:     name,
					SheetType: "GRID",
					GridProperties: &sheets.GridProperties{
						ColumnCount: cols,
						RowCount:    rows,
					},
				},
			},
		},
	}
	_, err := s.Service.Spreadsheets.BatchUpdate(s.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{Requests: reqs}).Do()
	if err != nil {
		return nil, err
	}
	err = s.Reload()
	if err != nil {
		return nil, err
	}
	return s.SheetByName(name)
}

func (s *Sheet) Id() int64 {
	return s.Properties.SheetId
}

func (s *Sheet) RowCount() int64 {
	return s.Properties.GridProperties.RowCount
}

func (s *Sheet) ColumnCount() int64 {
	return s.Properties.GridProperties.ColumnCount
}

func (s *Sheet) Read(range_ string) ([][]interface{}, error) {
	resp, err := s.Spreadsheet.Service.Spreadsheets.Values.Get(s.Spreadsheet.SpreadsheetId, s.Properties.Title+"!"+range_).Do()
	if err != nil {
		return nil, err
	}
	return resp.Values, nil
}

func (s *Sheet) Cell(cell string) (interface{}, error) {
	values, err := s.Read(cell)
	if err != nil {
		return nil, err
	}
	return values[0][0], nil
}

func (s *Sheet) Update(range_ string, values [][]interface{}) error {
	vr := &sheets.ValueRange{
		MajorDimension: "ROWS",
		Values:         values,
	}
	_, err := s.Spreadsheet.Service.Spreadsheets.Values.Update(s.Spreadsheet.SpreadsheetId, s.Properties.Title+"!"+range_, vr).ValueInputOption("USER_ENTERED").Do()
	return err
}

func (s *Sheet) Resize(cols, rows int64) error {
	reqs := []*sheets.Request{
		&sheets.Request{
			UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
				Properties: &sheets.SheetProperties{
					SheetId: s.Properties.SheetId,
					GridProperties: &sheets.GridProperties{
						ColumnCount: cols,
						RowCount:    rows,
					},
				},
				Fields: "gridProperties(rowCount,columnCount)",
			},
		},
	}
	_, err := s.Spreadsheet.Service.Spreadsheets.BatchUpdate(s.Spreadsheet.SpreadsheetId, &sheets.BatchUpdateSpreadsheetRequest{Requests: reqs}).Do()
	if err != nil {
		return err
	}
	s.Properties.GridProperties.ColumnCount = cols
	s.Properties.GridProperties.RowCount = rows
	return nil
}

func (s *Sheet) Append(values [][]interface{}) error {
	start := s.RowCount() + 1
	err := s.Resize(s.ColumnCount(), s.RowCount()+int64(len(values)))
	if err != nil {
		return err
	}
	return s.Update(fmt.Sprintf("A%d", start), values)
}
