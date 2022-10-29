package httphandler

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/adetunjii/google-sheets-connector/internal/model"
	"github.com/adetunjii/google-sheets-connector/internal/services/google"
	"github.com/adetunjii/google-sheets-connector/pkg/logger"
	"github.com/adetunjii/google-sheets-connector/pkg/utils/httputils"
)

type Handler struct {
	googleClient      *google.GoogleClient
	googleSheetClient *google.GoogleSheetClient
	logger            logger.AppLogger
}

func New(googleClient *google.GoogleClient, logger logger.AppLogger) *Handler {
	return &Handler{
		googleClient: googleClient,
		logger:       logger,
	}
}

func (h *Handler) OauthGoogle(w http.ResponseWriter, r *http.Request) {
	rw := httputils.NewResponseWriter(w)

	url, err := h.googleClient.HandleGoogleLogin()
	if err != nil {
		rw.Error(err, http.StatusBadRequest)
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) OauthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	rw := httputils.NewResponseWriter(w)

	code := r.FormValue("code")
	errorReason := r.FormValue("error_reason")

	resp, err := h.googleClient.HandleGoogleCallback(code, errorReason)
	if err != nil {
		rw.Error(err, http.StatusBadRequest)
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		rw.Error(err, http.StatusInternalServerError)
	}

	rw.WriteJSON(bytes)
}

func (h *Handler) CreateGoogleSheet(w http.ResponseWriter, r *http.Request) {
	rw := httputils.NewResponseWriter(w)

	spreadSheet := &google.SpreadSheet{}

	if err := json.NewDecoder(r.Body).Decode(spreadSheet); err != nil {
		rw.Error(err, http.StatusBadRequest)
	}

	// create new client based on the token sent and the sheet title
	googleSheetClient := google.NewGoogleSheetClient(h.googleClient, spreadSheet.Token, h.logger)
	h.googleSheetClient = googleSheetClient

	// create a spreadsheet
	s, err := googleSheetClient.CreateSpreadSheet(spreadSheet.Title)
	if err != nil {
		rw.Error(err, http.StatusBadRequest)
		return
	}

	spreadSheet.ID = s.SpreadsheetId
	spreadSheet.Url = s.SpreadsheetUrl
	sheetID := s.Sheets[0].Properties.SheetId

	sheetTitle := s.Sheets[0].Properties.Title

	headers := getSheetHeaders()

	// create column headers
	if err := googleSheetClient.AppendColumnHeaders(spreadSheet.ID, sheetID, sheetTitle, headers); err != nil {
		rw.Error(err, http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(spreadSheet)
	if err != nil {
		rw.Error(err, http.StatusInternalServerError)
		return
	}
	rw.WriteJSON(bytes)
}

func (h *Handler) writeToSheet(w http.ResponseWriter, r *http.Request) {
	rw := httputils.NewResponseWriter(w)

	spreadsheetID := "1tizuK0aeRmhYgJS_3IWCCHq5tbhMvvd2zk303H3Y_Lo"
	qd := &model.QuestionnarieData{}

	if err := json.NewDecoder(r.Body).Decode(qd); err != nil {
		rw.Error(err, http.StatusBadRequest)
	}

	if err := h.googleSheetClient.WriteToSheet(spreadsheetID, qd); err != nil {
		rw.Error(err, http.StatusBadRequest)
		return
	}

	bytes, err := json.Marshal(qd)
	if err != nil {
		rw.Error(err, http.StatusInternalServerError)
		return
	}

	rw.WriteJSON(bytes)
}

func getSheetHeaders() []string {
	t := reflect.TypeOf(model.QuestionnarieData{})
	headers := make([]string, t.NumField())
	for i := range headers {
		headers[i] = t.Field(i).Name
	}

	return headers
}
