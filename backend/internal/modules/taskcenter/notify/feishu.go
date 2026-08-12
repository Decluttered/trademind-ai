package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type feishuRobotRequest struct {
	Timestamp string `json:"timestamp,omitempty"`
	Sign      string `json:"sign,omitempty"`
	MsgType   string `json:"msg_type"`
	Content   struct {
		Text string `json:"text"`
	} `json:"content"`
}

type feishuRobotResponse struct {
	Code          *int   `json:"code"`
	Message       string `json:"msg"`
	StatusCode    *int   `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`
}

// SendFeishu sends a native text message to a Feishu custom bot.
func SendFeishu(ctx context.Context, d FeishuDeps, payload AlertNotificationPayload) AlertNotificationResult {
	reqBody := feishuRobotRequest{MsgType: "text"}
	reqBody.Content.Text = buildRobotText(payload, 16*1024)
	if secret := strings.TrimSpace(d.Secret); secret != "" {
		reqBody.Timestamp = strconv.FormatInt(time.Now().Unix(), 10)
		reqBody.Sign = feishuRobotSignature(reqBody.Timestamp, secret)
	}

	httpRes := postRobotJSON(ctx, "feishu", d.URL, reqBody, d.Timeout, d.AllowHTTP)
	res := httpRes.AlertNotificationResult
	if res.Status != "" {
		return res
	}
	var vendor feishuRobotResponse
	if err := json.Unmarshal(httpRes.Body, &vendor); err != nil {
		res.Status = "failed"
		res.ErrorMessage = "invalid feishu response"
		return res
	}
	code, message, ok := feishuVendorResult(vendor)
	if !ok {
		res.Status = "failed"
		res.ErrorMessage = "invalid feishu response"
		return res
	}
	message = sanitizeRobotMessage(message, d.URL, d.Secret)
	setRobotVendorSummary(&res, code, message)
	if code != 0 {
		res.Status = "failed"
		res.ErrorMessage = truncateStr(fmt.Sprintf("feishu code %d: %s", code, message), 500)
		return res
	}
	res.Status = "success"
	return res
}

func feishuRobotSignature(timestamp, secret string) string {
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	_, _ = mac.Write(nil)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func feishuVendorResult(v feishuRobotResponse) (int, string, bool) {
	if v.Code != nil {
		return *v.Code, v.Message, true
	}
	if v.StatusCode != nil {
		return *v.StatusCode, v.StatusMessage, true
	}
	return 0, "", false
}
