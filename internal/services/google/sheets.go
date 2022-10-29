package google

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/adetunjii/google-sheets-connector/internal/model"
	"github.com/adetunjii/google-sheets-connector/pkg/logger"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	VALUE_INPUT_OPTION = "RAW"
	INSERT_DATA_OPTION = "INSERT_ROWS"
)

type SpreadSheet struct {
	ID     string          `json:"id"`
	Sheets []*sheets.Sheet `json:"sheets"`
	Title  string          `json:"title"`
	Url    string          `json:"url"`
	Token  *oauth2.Token   `json:"token"`
}

type GoogleSheetClient struct {
	svc    *sheets.Service
	logger logger.AppLogger
}

func NewGoogleSheetClient(googleClient *GoogleClient, token *oauth2.Token, logger logger.AppLogger) *GoogleSheetClient {
	client := googleClient.config.Client(context.Background(), token)

	svc, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil
	}

	return &GoogleSheetClient{
		svc:    svc,
		logger: logger,
	}
}

func (gs *GoogleSheetClient) CreateSpreadSheet(name string) (*sheets.Spreadsheet, error) {
	spreadsheet, err := gs.svc.Spreadsheets.Create(&sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: name,
		},
	}).Do()

	if err != nil {
		return nil, err
	}

	return spreadsheet, nil
}

func (gs *GoogleSheetClient) CreateSheet(spreadsheetId string, name string, columnCount int64) (*int64, error) {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: name,
					},
				},
			},
		},
	}

	resp, err := gs.svc.Spreadsheets.BatchUpdate(spreadsheetId, req).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}

	sheetresp := resp.Replies[0].AddSheet
	sheetId := sheetresp.Properties.SheetId

	return &sheetId, nil
}

func (gs *GoogleSheetClient) DeleteSheet(spreadSheetID string, sheetID int64) error {
	updateSheetRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteSheet: &sheets.DeleteSheetRequest{
					SheetId: sheetID,
				},
			},
		},
	}

	_, err := gs.svc.Spreadsheets.BatchUpdate(spreadSheetID, updateSheetRequest).Context(context.Background()).Do()
	if err != nil {
		return err
	}

	return nil
}

func (gs *GoogleSheetClient) AppendColumnHeaders(spreadSheetId string, sheetID int64, sheetTitle string, headers []string) error {

	h := make([]interface{}, len(headers))

	// value range values
	v := [][]interface{}{}

	sort.Strings(headers)

	for i, header := range headers {
		h[i] = strings.ToUpper(header)
	}

	v = append(v, h)

	values := &sheets.ValueRange{
		Values: v,
	}

	// cellRange, valueRange
	cellRange := fmt.Sprintf("%s!R1C1:R1C%d", sheetTitle, len(headers))
	// append data to the sheet
	if err := gs.appendRowData(spreadSheetId, cellRange, values); err != nil {
		return err
	}

	headerFormat := &sheets.CellFormat{
		VerticalAlignment:   "MIDDLE",
		HorizontalAlignment: "CENTER",
		TextFormat: &sheets.TextFormat{
			Bold: true,
		},
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				RepeatCell: &sheets.RepeatCellRequest{

					Range: &sheets.GridRange{
						SheetId:          sheetID,
						StartRowIndex:    0,
						EndRowIndex:      1,
						StartColumnIndex: 0,
						EndColumnIndex:   int64(len(headers)),
					},
					Cell: &sheets.CellData{
						UserEnteredFormat: headerFormat,
					},
					Fields: "userEnteredFormat(verticalAlignment, horizontalAlignment, textFormat)",
				},
			},
			{

				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{
						SheetId: sheetID,
						GridProperties: &sheets.GridProperties{
							FrozenRowCount: 1,
						},
					},
					Fields: "gridProperties.frozenRowCount",
				},
			},
		},
	}

	_, err := gs.svc.Spreadsheets.BatchUpdate(spreadSheetId, req).Context(context.Background()).Do()
	if err != nil {
		return err
	}

	return nil
}

func (gs *GoogleSheetClient) RowCount(spreadSheetID string, cellRange string) int {
	valueRange, err := gs.svc.Spreadsheets.Values.Get(spreadSheetID, cellRange).Context(context.Background()).Do()
	if err != nil {
		gs.logger.Error("failed to fetch values :: stacktrace ::", err)
	}
	return len(valueRange.Values)
}

// uses R1C1 notation for cell ranges
func (gs *GoogleSheetClient) appendRowData(spreadSheetID string, cellRange string, rowValues *sheets.ValueRange) error {
	fmt.Println(rowValues)
	_, err := gs.svc.Spreadsheets.Values.Append(spreadSheetID, cellRange, rowValues).ValueInputOption(VALUE_INPUT_OPTION).InsertDataOption(INSERT_DATA_OPTION).Context(context.Background()).Do()
	if err != nil {
		gs.logger.Error("failed to append row data :: stacktrace :: ", err)
		return err
	}
	return nil
}

func (gs *GoogleSheetClient) WriteToSheet(spreadSheetID string, data *model.QuestionnarieData) error {

	if err := data.Validate(); err != nil {
		return err
	}

	v := [][]interface{}{}
	row := []interface{}{}

	m := map[string]interface{}{}

	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &m); err != nil {
		return err
	}

	keys := []string{}

	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		row = append(row, m[key])
	}

	v = append(v, row)

	valueRange := &sheets.ValueRange{
		Values: v,
	}

	fmt.Println("here")
	if err := gs.appendRowData(spreadSheetID, "Sheet1", valueRange); err != nil {
		return err
	}

	return nil
}
