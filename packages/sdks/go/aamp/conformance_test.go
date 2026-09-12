package aamp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type parserConformanceInput struct {
	From      string            `json:"from"`
	To        string            `json:"to"`
	MessageID string            `json:"messageId"`
	Subject   string            `json:"subject"`
	Headers   map[string]string `json:"headers"`
	BodyText  string            `json:"bodyText"`
}

type parserConformanceCase struct {
	Name     string                 `json:"name"`
	Input    parserConformanceInput `json:"input"`
	Expected map[string]any         `json:"expected"`
}

type parserConformanceSuite struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Cases         []parserConformanceCase `json:"cases"`
}

func parsedMessageFields(message *ParsedMessage) map[string]any {
	return map[string]any{
		"protocolVersion":      message.ProtocolVersion,
		"intent":               message.Intent,
		"taskId":               message.TaskID,
		"title":                message.Title,
		"priority":             message.Priority,
		"expiresAt":            message.ExpiresAt,
		"sessionKey":           message.SessionKey,
		"dispatchContext":      message.DispatchContext,
		"dispatchContextRules": message.DispatchContextRules,
		"parentTaskId":         message.ParentTaskID,
		"from":                 message.From,
		"to":                   message.To,
		"messageId":            message.MessageID,
		"subject":              message.Subject,
		"bodyText":             message.BodyText,
		"status":               message.Status,
		"success":              message.Success,
		"reason":               message.ErrorMsg,
		"output":               message.Output,
		"errorMsg":             message.ErrorMsg,
		"structuredResult":     message.StructuredResult,
		"question":             message.Question,
		"blockedReason":        message.BlockedReason,
		"suggestedOptions":     message.SuggestedOptions,
		"streamId":             message.StreamID,
		"summary":              message.Summary,
		"pairCode":             message.PairCode,
	}
}

func normalizeConformanceJSON(t *testing.T, value any) any {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal conformance value: %v", err)
	}
	var normalized any
	if err := json.Unmarshal(payload, &normalized); err != nil {
		t.Fatalf("unmarshal conformance value: %v", err)
	}
	return normalized
}

func TestSharedParserConformanceFixtures(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "..", "..", "conformance", "fixtures", "parser.json")
	payload, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read parser fixtures: %v", err)
	}

	var fixtures parserConformanceSuite
	if err := json.Unmarshal(payload, &fixtures); err != nil {
		t.Fatalf("decode parser fixtures: %v", err)
	}
	if fixtures.SchemaVersion == "" {
		t.Fatal("parser fixtures must declare schemaVersion")
	}

	for _, fixture := range fixtures.Cases {
		fixture := fixture
		t.Run(fixture.Name, func(t *testing.T) {
			message, err := ParseAampHeaders(EmailMetadata{
				From:      fixture.Input.From,
				To:        fixture.Input.To,
				MessageID: fixture.Input.MessageID,
				Subject:   fixture.Input.Subject,
				Headers:   fixture.Input.Headers,
				BodyText:  fixture.Input.BodyText,
			})
			if err != nil {
				t.Fatalf("parse fixture: %v", err)
			}
			if message == nil {
				t.Fatal("expected parsed message")
			}

			fields := parsedMessageFields(message)
			actual := make(map[string]any, len(fixture.Expected))
			for key := range fixture.Expected {
				actual[key] = fields[key]
			}

			if !reflect.DeepEqual(
				normalizeConformanceJSON(t, actual),
				normalizeConformanceJSON(t, fixture.Expected),
			) {
				t.Fatalf("canonical parsed message mismatch\nactual:   %#v\nexpected: %#v", actual, fixture.Expected)
			}
		})
	}
}
