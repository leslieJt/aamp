package aamp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"regexp"
	"strings"
)

const (
	AAMPProtocolVersion = "1.1"
	HeaderVersion       = "X-AAMP-Version"
	HeaderIntent        = "X-AAMP-Intent"
	HeaderTaskID        = "X-AAMP-TaskId"
	HeaderDispatchCtx   = "X-AAMP-Dispatch-Context"
	HeaderPriority      = "X-AAMP-Priority"
	HeaderExpiresAt     = "X-AAMP-Expires-At"
	HeaderStatus        = "X-AAMP-Status"
	HeaderErrorMsg      = "X-AAMP-ErrorMsg"
	HeaderStructured    = "X-AAMP-StructuredResult"
	HeaderSuggested     = "X-AAMP-SuggestedOptions"
	HeaderStreamID      = "X-AAMP-Stream-Id"
	HeaderParentTaskID  = "X-AAMP-ParentTaskId"
	HeaderCardSummary   = "X-AAMP-Card-Summary"
	HeaderSessionKey    = "X-AAMP-Session-Key"
	HeaderPairCode      = "X-AAMP-Pair-Code"
	HeaderDispatchRules = "X-AAMP-Dispatch-Context-Rules"
)

var dispatchContextKeyRE = regexp.MustCompile(`^[a-z0-9_-]+$`)

func NormalizeHeaders(headers map[string]string) map[string]string {
	normalized := make(map[string]string, len(headers))
	for key, value := range headers {
		normalized[strings.ToLower(key)] = value
	}
	return normalized
}

func ParseDispatchContextHeader(value string) map[string]string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	context := map[string]string{}
	for _, part := range strings.Split(value, ";") {
		segment := strings.TrimSpace(part)
		if segment == "" {
			continue
		}
		pieces := strings.SplitN(segment, "=", 2)
		if len(pieces) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(pieces[0]))
		if !dispatchContextKeyRE.MatchString(key) {
			continue
		}
		decoded, err := url.QueryUnescape(strings.TrimSpace(pieces[1]))
		if err != nil {
			context[key] = strings.TrimSpace(pieces[1])
			continue
		}
		context[key] = decoded
	}
	if len(context) == 0 {
		return nil
	}
	return context
}

func SerializeDispatchContextHeader(context map[string]string) string {
	if len(context) == 0 {
		return ""
	}
	parts := make([]string, 0, len(context))
	for rawKey, rawValue := range context {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		value := strings.TrimSpace(rawValue)
		if value == "" || !dispatchContextKeyRE.MatchString(key) {
			continue
		}
		parts = append(parts, key+"="+url.QueryEscape(value))
	}
	return strings.Join(parts, "; ")
}

func BuildDispatchHeaders(taskID, priority, expiresAt, sessionKey string, dispatchContext map[string]string, parentTaskID string) map[string]string {
	headers := map[string]string{
		HeaderVersion: AAMPProtocolVersion,
		HeaderIntent:  "task.dispatch",
		HeaderTaskID:  taskID,
		HeaderPriority: func() string {
			if priority == "" {
				return "normal"
			}
			return priority
		}(),
	}
	if expiresAt != "" {
		headers[HeaderExpiresAt] = expiresAt
	}
	if trimmed := strings.TrimSpace(sessionKey); trimmed != "" {
		headers[HeaderSessionKey] = trimmed
	}
	if serialized := SerializeDispatchContextHeader(dispatchContext); serialized != "" {
		headers[HeaderDispatchCtx] = serialized
	}
	if parentTaskID != "" {
		headers[HeaderParentTaskID] = parentTaskID
	}
	return headers
}

func BuildCancelHeaders(taskID string) map[string]string {
	return map[string]string{
		HeaderVersion: AAMPProtocolVersion,
		HeaderIntent:  "task.cancel",
		HeaderTaskID:  taskID,
	}
}

func BuildAckHeaders(taskID string) map[string]string {
	return map[string]string{
		HeaderVersion: AAMPProtocolVersion,
		HeaderIntent:  "task.ack",
		HeaderTaskID:  taskID,
	}
}

func BuildStreamOpenedHeaders(taskID, streamID string) map[string]string {
	return map[string]string{
		HeaderVersion:  AAMPProtocolVersion,
		HeaderIntent:   "task.stream.opened",
		HeaderTaskID:   taskID,
		HeaderStreamID: streamID,
	}
}

func BuildResultHeaders(taskID, status, errorMsg string, structuredResult []StructuredResultField) map[string]string {
	headers := map[string]string{
		HeaderVersion: AAMPProtocolVersion,
		HeaderIntent:  "task.result",
		HeaderTaskID:  taskID,
		HeaderStatus:  status,
	}
	if errorMsg != "" {
		headers[HeaderErrorMsg] = errorMsg
	}
	if encoded, err := encodeStructuredResult(structuredResult); err == nil && encoded != "" {
		headers[HeaderStructured] = encoded
	}
	return headers
}

func BuildHelpHeaders(taskID string, suggestedOptions []string) map[string]string {
	return map[string]string{
		HeaderVersion:   AAMPProtocolVersion,
		HeaderIntent:    "task.help_needed",
		HeaderTaskID:    taskID,
		HeaderSuggested: strings.Join(suggestedOptions, "|"),
	}
}

func BuildCardQueryHeaders(taskID string) map[string]string {
	return map[string]string{
		HeaderVersion: AAMPProtocolVersion,
		HeaderIntent:  "card.query",
		HeaderTaskID:  taskID,
	}
}

func BuildCardResponseHeaders(taskID, summary string) map[string]string {
	return map[string]string{
		HeaderVersion:     AAMPProtocolVersion,
		HeaderIntent:      "card.response",
		HeaderTaskID:      taskID,
		HeaderCardSummary: summary,
	}
}

func validatePairCode(pairCode string) (string, error) {
	trimmed := strings.TrimSpace(pairCode)
	if trimmed == "" {
		return "", fmt.Errorf("pairCode cannot be empty")
	}
	if strings.ContainsAny(trimmed, "\r\n") {
		return "", fmt.Errorf("pairCode contains invalid characters")
	}
	if !isValidBase64URLToken(trimmed) {
		return "", fmt.Errorf("pairCode must be a base64url token")
	}
	return trimmed, nil
}

func isValidBase64URLToken(value string) bool {
	if _, err := base64.RawURLEncoding.DecodeString(value); err == nil {
		return true
	}
	_, err := base64.URLEncoding.DecodeString(value)
	return err == nil
}

func BuildPairRequestHeaders(taskID, pairCode string, dispatchContextRules map[string][]string) (map[string]string, error) {
	validated, err := validatePairCode(pairCode)
	if err != nil {
		return nil, err
	}
	rules := dispatchContextRules
	if rules == nil {
		rules = map[string][]string{}
	}
	return map[string]string{
		HeaderVersion:       AAMPProtocolVersion,
		HeaderIntent:        "pair.request",
		HeaderTaskID:        taskID,
		HeaderPairCode:      validated,
		HeaderDispatchRules: encodeBase64URLJSON(rules),
	}, nil
}

func BuildPairRespondHeaders(taskID string, success bool, reason string) map[string]string {
	status := "rejected"
	if success {
		status = "completed"
	}
	headers := map[string]string{
		HeaderVersion: AAMPProtocolVersion,
		HeaderIntent:  "pair.respond",
		HeaderTaskID:  taskID,
		HeaderStatus:  status,
	}
	if !success {
		if cleaned := sanitizePairReason(reason); cleaned != "" {
			headers[HeaderErrorMsg] = cleaned
		}
	}
	return headers
}

func sanitizePairReason(reason string) string {
	cleaned := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(reason, "\r", " "), "\n", " "))
	return cleaned
}

func encodeBase64URLJSON(value any) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeBase64URLJSONMapStringSlice(value string) map[string][]string {
	if strings.TrimSpace(value) == "" {
		return map[string][]string{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		// Accept standard base64 with padding, matching Node's decode path.
		normalized := strings.ReplaceAll(strings.ReplaceAll(value, "-", "+"), "_", "/")
		switch len(normalized) % 4 {
		case 2:
			normalized += "=="
		case 3:
			normalized += "="
		}
		payload, err = base64.StdEncoding.DecodeString(normalized)
		if err != nil {
			return map[string][]string{}
		}
	}
	decoded := map[string][]string{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return map[string][]string{}
	}
	if decoded == nil {
		return map[string][]string{}
	}
	return decoded
}

func ParseAampHeaders(meta EmailMetadata) (*ParsedMessage, error) {
	headers := NormalizeHeaders(meta.Headers)
	intent := headers[strings.ToLower(HeaderIntent)]
	taskID := headers[strings.ToLower(HeaderTaskID)]
	if intent == "" || taskID == "" {
		return nil, nil
	}

	protocolVersion := headers[strings.ToLower(HeaderVersion)]
	if protocolVersion == "" {
		protocolVersion = AAMPProtocolVersion
	}

	subject, err := decodeSubject(meta.Subject)
	if err != nil {
		subject = meta.Subject
	}

	base := &ParsedMessage{
		ProtocolVersion: protocolVersion,
		Intent:          intent,
		TaskID:          taskID,
		From:            strings.Trim(meta.From, "<>"),
		To:              strings.Trim(meta.To, "<>"),
		MessageID:       meta.MessageID,
		Subject:         subject,
	}

	switch intent {
	case "task.dispatch":
		base.Title = strings.TrimSpace(strings.TrimPrefix(subject, "[AAMP Task]"))
		base.Priority = firstNonEmpty(headers[strings.ToLower(HeaderPriority)], "normal")
		base.ExpiresAt = headers[strings.ToLower(HeaderExpiresAt)]
		base.SessionKey = headers[strings.ToLower(HeaderSessionKey)]
		base.DispatchContext = ParseDispatchContextHeader(headers[strings.ToLower(HeaderDispatchCtx)])
		base.ParentTaskID = headers[strings.ToLower(HeaderParentTaskID)]
		base.BodyText = normalizeBodyText(meta.BodyText)
	case "task.cancel":
		base.BodyText = normalizeBodyText(meta.BodyText)
	case "task.result":
		body := parseTaskResultBody(meta.BodyText)
		base.Status = firstNonEmpty(headers[strings.ToLower(HeaderStatus)], "completed")
		base.Output = body.Output
		base.ErrorMsg = firstNonEmpty(body.ErrorMsg, headers[strings.ToLower(HeaderErrorMsg)])
		base.StructuredResult, _ = decodeStructuredResult(headers[strings.ToLower(HeaderStructured)])
	case "task.help_needed":
		body := parseTaskHelpBody(meta.BodyText)
		base.Question = body.Question
		base.BlockedReason = body.BlockedReason
		base.SuggestedOptions = body.SuggestedOptions
	case "task.ack":
	case "task.stream.opened":
		base.StreamID = headers[strings.ToLower(HeaderStreamID)]
	case "pair.request":
		pairCode := headers[strings.ToLower(HeaderPairCode)]
		if pairCode == "" {
			return nil, nil
		}
		base.PairCode = pairCode
		base.DispatchContextRules = decodeBase64URLJSONMapStringSlice(headers[strings.ToLower(HeaderDispatchRules)])
		base.BodyText = normalizeBodyText(meta.BodyText)
	case "pair.respond":
		rawStatus := headers[strings.ToLower(HeaderStatus)]
		if rawStatus == "completed" {
			base.Status = "completed"
			base.Success = true
		} else {
			base.Status = "rejected"
			base.Success = false
		}
		base.ErrorMsg = headers[strings.ToLower(HeaderErrorMsg)]
		base.BodyText = normalizeBodyText(meta.BodyText)
	case "card.query":
		base.BodyText = normalizeBodyText(meta.BodyText)
	case "card.response":
		base.Summary = headers[strings.ToLower(HeaderCardSummary)]
		base.BodyText = normalizeBodyText(meta.BodyText)
	}

	return base, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func encodeStructuredResult(value []StructuredResultField) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeStructuredResult(value string) (any, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

type resultBody struct {
	Output   string
	ErrorMsg string
}

func parseTaskResultBody(body string) resultBody {
	normalized := normalizeBodyText(body)
	if normalized == "" {
		return resultBody{}
	}
	output := extractBodySection(normalized, "Output", []string{"Error"})
	errorMsg := extractBodySection(normalized, "Error", nil)
	if output != "" || errorMsg != "" {
		return resultBody{Output: output, ErrorMsg: errorMsg}
	}
	return resultBody{Output: normalized}
}

type helpBody struct {
	Question         string
	BlockedReason    string
	SuggestedOptions []string
}

func parseTaskHelpBody(body string) helpBody {
	normalized := normalizeBodyText(body)
	if normalized == "" {
		return helpBody{}
	}
	question := extractBodySection(normalized, "Question", []string{"Blocked reason", "Suggested options"})
	blockedReason := extractBodySection(normalized, "Blocked reason", []string{"Suggested options"})
	suggested := parseSuggestedOptions(extractBodySection(normalized, "Suggested options", nil))
	if question != "" || blockedReason != "" || len(suggested) > 0 {
		return helpBody{Question: question, BlockedReason: blockedReason, SuggestedOptions: suggested}
	}
	return helpBody{Question: normalized}
}

func normalizeBodyText(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
}

func extractBodySection(bodyText, label string, nextLabels []string) string {
	if bodyText == "" {
		return ""
	}
	normalized := "\n" + bodyText
	lower := strings.ToLower(normalized)
	prefix := "\n" + strings.ToLower(label) + ":"
	start := strings.Index(lower, prefix)
	if start < 0 {
		return ""
	}
	contentStart := start + len(prefix)
	for contentStart < len(normalized) {
		current := normalized[contentStart]
		if current == ' ' || current == '\t' {
			contentStart++
			continue
		}
		break
	}

	end := len(normalized)
	for _, nextLabel := range nextLabels {
		candidate := strings.Index(lower[contentStart:], "\n"+strings.ToLower(nextLabel)+":")
		if candidate >= 0 {
			absolute := contentStart + candidate
			if absolute < end {
				end = absolute
			}
		}
	}

	return strings.TrimSpace(normalized[contentStart:end])
}

func parseSuggestedOptions(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	re := regexp.MustCompile(`^\s*(?:[-*]|\d+\.)\s*`)
	lines := []string{}
	for _, line := range strings.Split(value, "\n") {
		cleaned := strings.TrimSpace(re.ReplaceAllString(line, ""))
		if cleaned != "" {
			lines = append(lines, cleaned)
		}
	}
	return lines
}

func decodeSubject(value string) (string, error) {
	decoder := new(mime.WordDecoder)
	return decoder.DecodeHeader(value)
}
