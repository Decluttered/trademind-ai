package notify

import (
	"context"
	"encoding/json"
	"fmt"
)

type weComRobotRequest struct {
	MsgType string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

type weComRobotResponse struct {
	ErrorCode    *int   `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
}

// SendWeCom sends a native text message to an Enterprise WeChat group robot.
func SendWeCom(ctx context.Context, d WeComDeps, payload AlertNotificationPayload) AlertNotificationResult {
	reqBody := weComRobotRequest{MsgType: "text"}
	reqBody.Text.Content = buildRobotText(payload, 1900)

	httpRes := postRobotJSON(ctx, "wecom", d.URL, reqBody, d.Timeout, d.AllowHTTP)
	res := httpRes.AlertNotificationResult
	if res.Status != "" {
		return res
	}
	var vendor weComRobotResponse
	if err := json.Unmarshal(httpRes.Body, &vendor); err != nil || vendor.ErrorCode == nil {
		res.Status = "failed"
		res.ErrorMessage = "invalid wecom response"
		return res
	}
	vendor.ErrorMessage = sanitizeRobotMessage(vendor.ErrorMessage, d.URL)
	setRobotVendorSummary(&res, *vendor.ErrorCode, vendor.ErrorMessage)
	if *vendor.ErrorCode != 0 {
		res.Status = "failed"
		res.ErrorMessage = truncateStr(fmt.Sprintf("wecom code %d: %s", *vendor.ErrorCode, vendor.ErrorMessage), 500)
		return res
	}
	res.Status = "success"
	return res
}
