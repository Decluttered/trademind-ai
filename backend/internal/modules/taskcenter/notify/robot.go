package notify

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type robotHTTPResult struct {
	AlertNotificationResult
	Body []byte
}

func postRobotJSON(
	ctx context.Context,
	channel string,
	rawURL string,
	body any,
	timeout time.Duration,
	allowHTTP bool,
) robotHTTPResult {
	res := robotHTTPResult{AlertNotificationResult: AlertNotificationResult{Channel: channel}}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		res.Status = "skipped"
		res.ErrorMessage = channel + " webhook url empty"
		return res
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		res.Status = "failed"
		res.ErrorMessage = "invalid " + channel + " webhook url"
		return res
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
	case "http":
		if !allowHTTP {
			res.Status = "skipped"
			res.ErrorMessage = "http not allowed"
			res.Target = maskRobotTarget(u)
			return res
		}
	default:
		res.Status = "failed"
		res.ErrorMessage = "only http/https"
		return res
	}
	res.Target = maskRobotTarget(u)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		res.Status = "failed"
		res.ErrorMessage = "encode " + channel + " request"
		return res
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(bodyBytes))
	if err != nil {
		res.Status = "failed"
		res.ErrorMessage = truncateStr(err.Error(), 400)
		return res
	}
	req.Header.Set("Content-Type", "application/json")

	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		res.Status = "failed"
		if ctx.Err() != nil {
			res.ErrorMessage = channel + " request canceled or timed out"
		} else {
			res.ErrorMessage = channel + " request failed"
		}
		return res
	}
	defer resp.Body.Close()
	res.RawSummary = map[string]any{"httpStatus": resp.StatusCode}
	res.Body, _ = io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		res.Status = "failed"
		res.ErrorMessage = fmt.Sprintf("http %d", resp.StatusCode)
	}
	return res
}

func buildRobotText(p AlertNotificationPayload, maxBytes int) string {
	var b strings.Builder
	b.WriteString("TradeMind 告警\n")
	b.WriteString("等级: " + strings.TrimSpace(p.Severity) + "\n")
	b.WriteString("标题: " + truncateStr(strings.TrimSpace(p.Title), 240) + "\n")
	if v := strings.TrimSpace(p.FailureCategory); v != "" {
		b.WriteString("分类: " + v + "\n")
	}
	if v := strings.TrimSpace(p.TaskType); v != "" {
		b.WriteString("任务: " + v + "\n")
	}
	if v := strings.TrimSpace(p.Message); v != "" {
		b.WriteString("摘要: " + truncateStr(v, 500) + "\n")
	}
	if v := strings.TrimSpace(p.SuggestedAction); v != "" {
		b.WriteString("建议: " + truncateStr(v, 400) + "\n")
	}
	if v := strings.TrimSpace(p.DetailURL); v != "" {
		b.WriteString("详情: " + v + "\n")
	}
	if v := strings.TrimSpace(p.OccurredAtRFC3339); v != "" {
		b.WriteString("时间(UTC): " + v)
	}
	return truncateUTF8Bytes(strings.TrimSpace(b.String()), maxBytes)
}

func truncateUTF8Bytes(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	const suffix = "..."
	limit := maxBytes - len(suffix)
	if limit <= 0 {
		return suffix[:maxBytes]
	}
	for limit > 0 && !utf8.ValidString(s[:limit]) {
		limit--
	}
	return s[:limit] + suffix
}

func setRobotVendorSummary(res *AlertNotificationResult, code int, message string) {
	if res.RawSummary == nil {
		res.RawSummary = map[string]any{}
	}
	res.RawSummary["vendorCode"] = code
	if message = strings.TrimSpace(message); message != "" {
		res.RawSummary["vendorMessage"] = truncateStr(message, 200)
	}
}

func sanitizeRobotMessage(message, rawURL string, secrets ...string) string {
	values := append([]string{strings.TrimSpace(rawURL)}, secrets...)
	if u, err := url.Parse(strings.TrimSpace(rawURL)); err == nil {
		for _, queryValues := range u.Query() {
			values = append(values, queryValues...)
		}
		pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(pathParts) > 0 && len(pathParts[len(pathParts)-1]) >= 8 {
			values = append(values, pathParts[len(pathParts)-1])
		}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if len(value) >= 4 {
			message = strings.ReplaceAll(message, value, "[redacted]")
		}
	}
	return truncateStr(strings.TrimSpace(message), 200)
}

func maskRobotTarget(u *url.URL) string {
	if u == nil {
		return ""
	}
	h := sha256.Sum256([]byte(u.Host + u.EscapedPath() + "?" + u.RawQuery))
	return fmt.Sprintf("%s#%s", u.Host, hex.EncodeToString(h[:6]))
}
