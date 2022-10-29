package model

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

type QuestionnarieData struct {
	FormID                *string   `json:"form_id"`
	QuestionID            *string   `json:"question_id"`
	RuleID                *string   `json:"rule_id"`
	QuestionTitle         *string   `json:"question_title"`
	QuestionRule          *string   `json:"question_rule"`
	IsRequired            *bool     `json:"is_required"`
	OptionID              *string   `json:"option_id"`
	Options               []*string `json:"options"`
	QuestionCreationDate  time.Time `json:"question_creation_date"`
	Answer                *string   `json:"answer"`
	SelectedAnswerOptions *string   `json:"selected_answer_option"`
	AnsweredOn            time.Time `json:"answered_on"`
	RespondentID          *string   `json:"respondent_id"`
	RespondentEmail       *string   `json:"respondent_email"`
	RespondentPhoneNumber *string   `json:"respondent_phone_number"`
	OrgID                 *string   `json:"org_id"`
	AnswerID              *string   `json:"answer_id"`
	FormStartDate         time.Time `json:"form_start_date"`
	FormEndDate           time.Time `json:"form_end_date"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type GoogleSheetKafkaMessage struct {
	SpreadSheetID string            `json:"spreadsheet_id"`
	SheetID       string            `json:"sheet_id"`
	Token         *oauth2.Token     `json:"token"`
	Questionnaire QuestionnarieData `json:"questionnaire"`
}

func (q *QuestionnarieData) Validate() error {
	if q.FormID == nil {
		return errors.New("form ID cannot be empty")
	}

	if q.QuestionID == nil {
		return errors.New("question ID cannot be empty")
	}

	if q.QuestionTitle == nil {
		return errors.New("question title cannot be empty")
	}

	if q.AnswerID == nil {
		return errors.New("answer cannot be empty")
	}

	if q.Answer == nil && q.SelectedAnswerOptions == nil {
		return errors.New("answer(s) cannot be empty")
	}

	if q.RespondentID == nil {
		return errors.New("respondent id cannot be empty")
	}

	if q.RespondentEmail == nil {
		return errors.New("respondent email cannot be empty")
	}

	if q.OrgID == nil {
		return errors.New("organization id cannot be empty")
	}

	if q.FormStartDate.IsZero() {
		return errors.New("form start date not specified")
	}

	if q.CreatedAt.IsZero() {
		return errors.New("form created at date not specified")
	}

	return nil
}

func (q *QuestionnarieData) ToInterface() interface{} {
	fmt.Println("conversion failure")
	var i interface{} = q
	return i
}
