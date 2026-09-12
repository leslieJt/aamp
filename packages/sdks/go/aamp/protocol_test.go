package aamp

import (
	"strings"
	"testing"
)

func TestDispatchContextRoundTrip(t *testing.T) {
	encoded := SerializeDispatchContextHeader(map[string]string{
		"project_key": "proj 123",
		"user_key":    "alice",
	})
	if encoded != "project_key=proj+123; user_key=alice" && encoded != "user_key=alice; project_key=proj+123" {
		t.Fatalf("unexpected encoded value: %s", encoded)
	}
	decoded := ParseDispatchContextHeader(encoded)
	if decoded["project_key"] != "proj 123" || decoded["user_key"] != "alice" {
		t.Fatalf("unexpected decoded payload: %#v", decoded)
	}
}

func TestDispatchSessionKeyRoundTrip(t *testing.T) {
	headers := BuildDispatchHeaders("task-1", "normal", "", "sess-1", nil, "")
	if headers[HeaderSessionKey] != "sess-1" {
		t.Fatalf("expected session key header to be set, got %q", headers[HeaderSessionKey])
	}
	message, err := ParseAampHeaders(EmailMetadata{
		From:      "dispatcher@example.com",
		To:        "agent@example.com",
		MessageID: "<msg-1@example.com>",
		Subject:   "[AAMP Task] Do something",
		Headers:   headers,
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if message == nil {
		t.Fatal("expected parsed message")
	}
	if message.SessionKey != "sess-1" {
		t.Fatalf("expected parsed session key %q, got %q", "sess-1", message.SessionKey)
	}
}

func TestDispatchSessionKeyEmpty(t *testing.T) {
	headers := BuildDispatchHeaders("task-1", "normal", "", "", nil, "")
	if _, ok := headers[HeaderSessionKey]; ok {
		t.Fatalf("expected no session key header for empty value")
	}
	message, err := ParseAampHeaders(EmailMetadata{
		From:    "dispatcher@example.com",
		To:      "agent@example.com",
		Subject: "[AAMP Task] Do something",
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if message == nil {
		t.Fatal("expected parsed message")
	}
	if message.SessionKey != "" {
		t.Fatalf("expected empty session key, got %q", message.SessionKey)
	}
}

func TestDispatchSessionKeyTrimmed(t *testing.T) {
	headers := BuildDispatchHeaders("task-1", "normal", "", "  sess-trim  ", nil, "")
	if headers[HeaderSessionKey] != "sess-trim" {
		t.Fatalf("expected trimmed session key %q, got %q", "sess-trim", headers[HeaderSessionKey])
	}
}

func TestParseTaskResult(t *testing.T) {
	headers := BuildResultHeaders("task-2", "completed", "", []StructuredResultField{
		{FieldKey: "summary", FieldTypeKey: "text", Value: "done"},
	})
	message, err := ParseAampHeaders(EmailMetadata{
		From:      "agent@example.com",
		To:        "dispatcher@example.com",
		MessageID: "<msg-2@example.com>",
		Subject:   "[AAMP Result] Task task-2 - completed",
		BodyText:  "Output:\ndone",
		Headers:   headers,
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if message == nil {
		t.Fatal("expected parsed message")
	}
	if message.Intent != "task.result" || message.Output != "done" || message.TaskID != "task-2" {
		t.Fatalf("unexpected parsed message: %#v", message)
	}
}

func TestPairRequestRoundTrip(t *testing.T) {
	rules := map[string][]string{"source": {"feishu", "github"}}
	headers, err := BuildPairRequestHeaders("pair-1", "abc123", rules)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	if headers[HeaderIntent] != "pair.request" {
		t.Fatalf("expected pair.request intent, got %q", headers[HeaderIntent])
	}
	if headers[HeaderPairCode] != "abc123" {
		t.Fatalf("expected pair code header, got %q", headers[HeaderPairCode])
	}
	if headers[HeaderDispatchRules] == "" {
		t.Fatal("expected dispatch context rules header")
	}

	message, err := ParseAampHeaders(EmailMetadata{
		From:      "bridge@example.com",
		To:        "agent@example.com",
		MessageID: "<pair-req-1@example.com>",
		Subject:   "[AAMP Pair] Connection request",
		BodyText:  "AAMP Pair Request",
		Headers:   headers,
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if message == nil {
		t.Fatal("expected parsed message")
	}
	if message.Intent != "pair.request" || message.TaskID != "pair-1" || message.PairCode != "abc123" {
		t.Fatalf("unexpected parsed pair request: %#v", message)
	}
	if got := message.DispatchContextRules["source"]; len(got) != 2 || got[0] != "feishu" || got[1] != "github" {
		t.Fatalf("unexpected dispatch context rules: %#v", message.DispatchContextRules)
	}
}

func TestPairRequestMissingCode(t *testing.T) {
	headers, err := BuildPairRequestHeaders("pair-missing", "code", nil)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	delete(headers, HeaderPairCode)
	message, err := ParseAampHeaders(EmailMetadata{
		From:    "bridge@example.com",
		To:      "agent@example.com",
		Subject: "[AAMP Pair] Connection request",
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if message != nil {
		t.Fatalf("expected nil message without pair code, got %#v", message)
	}
}

func TestPairRespondFailureRoundTrip(t *testing.T) {
	headers := BuildPairRespondHeaders("pair-rt-1", false, "invalid or expired pair code")
	if headers[HeaderStatus] != "rejected" {
		t.Fatalf("expected rejected status, got %q", headers[HeaderStatus])
	}
	if headers[HeaderErrorMsg] != "invalid or expired pair code" {
		t.Fatalf("expected error message header, got %q", headers[HeaderErrorMsg])
	}

	message, err := ParseAampHeaders(EmailMetadata{
		From:      "agent@example.com",
		To:        "app@example.com",
		MessageID: "<msg-pair-1@example.com>",
		Subject:   "[AAMP Pair] rejected",
		BodyText:  "AAMP Pair Response",
		Headers:   headers,
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if message == nil {
		t.Fatal("expected parsed message")
	}
	if message.Intent != "pair.respond" || message.TaskID != "pair-rt-1" {
		t.Fatalf("unexpected parsed pair respond: %#v", message)
	}
	if message.Success || message.Status != "rejected" || message.ErrorMsg != "invalid or expired pair code" {
		t.Fatalf("unexpected pair respond fields: %#v", message)
	}
}

func TestPairRespondSuccessOmitsErrorMsg(t *testing.T) {
	headers := BuildPairRespondHeaders("pair-ok", true, "should be ignored")
	if headers[HeaderStatus] != "completed" {
		t.Fatalf("expected completed status, got %q", headers[HeaderStatus])
	}
	if _, ok := headers[HeaderErrorMsg]; ok {
		t.Fatalf("expected no error message on successful pair respond")
	}

	message, err := ParseAampHeaders(EmailMetadata{
		From:    "agent@example.com",
		To:      "app@example.com",
		Subject: "[AAMP Pair] completed",
		Headers: headers,
	})
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if message == nil || !message.Success || message.Status != "completed" {
		t.Fatalf("unexpected success pair respond: %#v", message)
	}
}

func TestValidatePairCodeRejectsHeaderInjection(t *testing.T) {
	malicious := "abc\r\nX-AAMP-Intent: task.dispatch"
	_, err := validatePairCode(malicious)
	if err == nil {
		t.Fatal("expected error for injected pair code")
	}
}

func TestBuildPairRequestHeadersRejectsHeaderInjection(t *testing.T) {
	malicious := "abc\r\nX-AAMP-Intent: task.dispatch"
	_, err := BuildPairRequestHeaders("pair-1", malicious, nil)
	if err == nil {
		t.Fatal("expected error for injected pair code")
	}
}

func TestPairRequestMIMEHeaderInjectionRegression(t *testing.T) {
	malicious := "abc\r\nX-AAMP-Intent: task.dispatch"
	if _, err := BuildPairRequestHeaders("pair-1", malicious, nil); err == nil {
		t.Fatal("expected validation error before MIME build")
	}

	headers, err := BuildPairRequestHeaders("pair-1", "abc123", nil)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}
	mime := buildMIMEMessage("sender@example.com", "agent@example.com", "[AAMP Pair] Connection request", "body", "<msg@example.com>", "", headers, nil)
	if strings.Count(mime, "X-AAMP-Intent:") != 1 {
		t.Fatalf("expected exactly one X-AAMP-Intent header, got MIME:\n%s", mime)
	}
	if !strings.Contains(mime, "X-AAMP-Intent: pair.request") {
		t.Fatalf("expected pair.request intent in MIME, got:\n%s", mime)
	}
}

func TestSendPairRequestRejectsHeaderInjection(t *testing.T) {
	sender := &SmtpSender{User: "sender@example.com"}
	_, _, err := sender.SendPairRequest(SendPairRequestOptions{
		To:       "agent@example.com",
		PairCode: "abc\r\nX-AAMP-Intent: task.dispatch",
	})
	if err == nil {
		t.Fatal("expected SendPairRequest to reject injected pair code")
	}
}

func TestValidatePairCodeAcceptsBase64URL(t *testing.T) {
	for _, code := range []string{"abc123", "7r8g5A2k", "manual-code"} {
		validated, err := validatePairCode(code)
		if err != nil {
			t.Fatalf("expected %q to be accepted, got %v", code, err)
		}
		if validated != code {
			t.Fatalf("expected trimmed code %q, got %q", code, validated)
		}
	}
}

func TestValidatePairCodeRejectsInvalidToken(t *testing.T) {
	for _, code := range []string{"", "has space", "bad+plus", "bad/slash"} {
		if _, err := validatePairCode(code); err == nil {
			t.Fatalf("expected %q to be rejected", code)
		}
	}
}
