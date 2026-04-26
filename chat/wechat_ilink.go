// Copyright 2026 The Casibase Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chat

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	WeChatIlinkDefaultBaseUrl = "https://ilinkai.weixin.qq.com"
	WeChatIlinkDefaultBotType = "3"

	weChatIlinkAppId         = "bot"
	weChatIlinkClientVersion = "131335"
	weChatIlinkChannel       = "casibase"
	weChatIlinkMessageText   = 1
	weChatIlinkMessageBot    = 2
	weChatIlinkMessageFinish = 2
)

type WeChatIlinkClient struct {
	baseUrl    string
	token      string
	httpClient *http.Client
}

type WeChatIlinkQRCodeResponse struct {
	Qrcode             string `json:"qrcode"`
	QrcodeImageContent string `json:"qrcode_img_content"`
}

type WeChatIlinkQRCodeStatus struct {
	Status       string `json:"status"`
	BotToken     string `json:"bot_token"`
	IlinkBotId   string `json:"ilink_bot_id"`
	BaseUrl      string `json:"baseurl"`
	IlinkUserId  string `json:"ilink_user_id"`
	RedirectHost string `json:"redirect_host"`
}

type WeChatIlinkBaseInfo struct {
	ChannelVersion string `json:"channel_version"`
}

type WeChatIlinkGetUpdatesResponse struct {
	Ret                  int                   `json:"ret"`
	ErrCode              int                   `json:"errcode"`
	ErrMsg               string                `json:"errmsg"`
	Messages             []*WeChatIlinkMessage `json:"msgs"`
	GetUpdatesBuf        string                `json:"get_updates_buf"`
	LongPollingTimeoutMs int                   `json:"longpolling_timeout_ms"`
}

type WeChatIlinkMessage struct {
	Seq          int64                     `json:"seq"`
	MessageId    int64                     `json:"message_id"`
	FromUserId   string                    `json:"from_user_id"`
	ToUserId     string                    `json:"to_user_id"`
	ClientId     string                    `json:"client_id"`
	CreateTimeMs int64                     `json:"create_time_ms"`
	SessionId    string                    `json:"session_id"`
	MessageType  int                       `json:"message_type"`
	MessageState int                       `json:"message_state"`
	ItemList     []*WeChatIlinkMessageItem `json:"item_list"`
	ContextToken string                    `json:"context_token"`
}

type WeChatIlinkMessageItem struct {
	Type     int                  `json:"type"`
	TextItem *WeChatIlinkTextItem `json:"text_item,omitempty"`
}

type WeChatIlinkTextItem struct {
	Text string `json:"text"`
}

type weChatIlinkGetUpdatesRequest struct {
	GetUpdatesBuf string              `json:"get_updates_buf"`
	BaseInfo      WeChatIlinkBaseInfo `json:"base_info"`
}

type weChatIlinkSendMessageRequest struct {
	Message  WeChatIlinkMessage  `json:"msg"`
	BaseInfo WeChatIlinkBaseInfo `json:"base_info"`
}

type weChatIlinkApiResponse struct {
	Ret     int    `json:"ret"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type WeChatIlinkChatProvider struct {
	client *WeChatIlinkClient
}

func NewWeChatIlinkClient(baseUrl string, token string, httpClient *http.Client) *WeChatIlinkClient {
	if strings.TrimSpace(baseUrl) == "" {
		baseUrl = WeChatIlinkDefaultBaseUrl
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &WeChatIlinkClient{
		baseUrl:    strings.TrimRight(strings.TrimSpace(baseUrl), "/"),
		token:      strings.TrimSpace(token),
		httpClient: httpClient,
	}
}

func NewWeChatIlinkChatProvider(baseUrl string, token string, httpClient *http.Client) (*WeChatIlinkChatProvider, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("WeChat iLink bot token should not be empty")
	}

	return &WeChatIlinkChatProvider{
		client: NewWeChatIlinkClient(baseUrl, token, httpClient),
	}, nil
}

func (p *WeChatIlinkChatProvider) SendMessage(chatId string, text string) error {
	return p.client.SendTextMessage(chatId, "", text)
}

func (p *WeChatIlinkChatProvider) ParseWebhookRequest(body []byte) (*IncomingMessage, error) {
	return nil, fmt.Errorf("WeChat iLink Bot uses long polling and does not support webhook parsing")
}

func (p *WeChatIlinkChatProvider) SetWebhook(webhookUrl string) error {
	return fmt.Errorf("WeChat iLink Bot uses long polling and does not support webhook setup")
}

func (c *WeChatIlinkClient) StartQRCodeLogin() (*WeChatIlinkQRCodeResponse, error) {
	var response WeChatIlinkQRCodeResponse
	err := c.doGet(fmt.Sprintf("ilink/bot/get_bot_qrcode?bot_type=%s", url.QueryEscape(WeChatIlinkDefaultBotType)), &response)
	if err != nil {
		return nil, err
	}
	if response.Qrcode == "" || response.QrcodeImageContent == "" {
		return nil, fmt.Errorf("WeChat iLink get_bot_qrcode response is missing qrcode")
	}
	return &response, nil
}

func (c *WeChatIlinkClient) PollQRCodeStatus(qrcode string) (*WeChatIlinkQRCodeStatus, error) {
	if strings.TrimSpace(qrcode) == "" {
		return nil, fmt.Errorf("WeChat iLink qrcode should not be empty")
	}

	var response WeChatIlinkQRCodeStatus
	err := c.doGet(fmt.Sprintf("ilink/bot/get_qrcode_status?qrcode=%s", url.QueryEscape(qrcode)), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *WeChatIlinkClient) GetUpdatesWithContext(ctx context.Context, getUpdatesBuf string, timeoutMs int) (*WeChatIlinkGetUpdatesResponse, error) {
	request := weChatIlinkGetUpdatesRequest{
		GetUpdatesBuf: getUpdatesBuf,
		BaseInfo:      buildWeChatIlinkBaseInfo(),
	}

	var response WeChatIlinkGetUpdatesResponse
	err := c.doPost(ctx, "ilink/bot/getupdates", request, timeoutMs, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *WeChatIlinkClient) SendTextMessage(toUserId string, contextToken string, text string) error {
	if strings.TrimSpace(toUserId) == "" {
		return fmt.Errorf("WeChat iLink to_user_id should not be empty")
	}

	request := weChatIlinkSendMessageRequest{
		Message: WeChatIlinkMessage{
			ToUserId:     toUserId,
			ClientId:     generateWeChatIlinkClientId(),
			MessageType:  weChatIlinkMessageBot,
			MessageState: weChatIlinkMessageFinish,
			ContextToken: contextToken,
			ItemList: []*WeChatIlinkMessageItem{
				{
					Type:     weChatIlinkMessageText,
					TextItem: &WeChatIlinkTextItem{Text: text},
				},
			},
		},
		BaseInfo: buildWeChatIlinkBaseInfo(),
	}

	var response weChatIlinkApiResponse
	if err := c.doPost(context.Background(), "ilink/bot/sendmessage", request, 0, &response); err != nil {
		return err
	}
	if response.Ret != 0 || response.ErrCode != 0 {
		return fmt.Errorf("WeChat iLink sendmessage error: ret=%d errcode=%d errmsg=%s", response.Ret, response.ErrCode, response.ErrMsg)
	}
	return nil
}

func (m *WeChatIlinkMessage) Text() (string, bool) {
	if m == nil {
		return "", false
	}
	for _, item := range m.ItemList {
		if item != nil && item.Type == weChatIlinkMessageText && item.TextItem != nil && item.TextItem.Text != "" {
			return item.TextItem.Text, true
		}
	}
	return "", false
}

func (c *WeChatIlinkClient) doGet(endpoint string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.buildUrl(endpoint), nil)
	if err != nil {
		return err
	}
	setWeChatIlinkCommonHeaders(req.Header)
	return c.do(req, target)
}

func (c *WeChatIlinkClient) doPost(ctx context.Context, endpoint string, payload interface{}, timeoutMs int, target interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	var cancel context.CancelFunc
	if timeoutMs > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs+5000)*time.Millisecond)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildUrl(endpoint), bytes.NewReader(body))
	if err != nil {
		return err
	}
	setWeChatIlinkCommonHeaders(req.Header)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthorizationType", "ilink_bot_token")
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	req.Header.Set("X-WECHAT-UIN", randomWeChatIlinkUin())
	if c.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
	}

	return c.do(req, target)
}

func (c *WeChatIlinkClient) do(req *http.Request, target interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WeChat iLink API error (status %d): %s", resp.StatusCode, string(respBody))
	}
	if target == nil || len(respBody) == 0 {
		return nil
	}
	if err = json.Unmarshal(respBody, target); err != nil {
		return err
	}
	return nil
}

func (c *WeChatIlinkClient) buildUrl(endpoint string) string {
	endpoint = strings.TrimLeft(endpoint, "/")
	return fmt.Sprintf("%s/%s", c.baseUrl, endpoint)
}

func setWeChatIlinkCommonHeaders(header http.Header) {
	header.Set("iLink-App-Id", weChatIlinkAppId)
	header.Set("iLink-App-ClientVersion", weChatIlinkClientVersion)
}

func buildWeChatIlinkBaseInfo() WeChatIlinkBaseInfo {
	return WeChatIlinkBaseInfo{ChannelVersion: weChatIlinkChannel}
}

func randomWeChatIlinkUin() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return base64.StdEncoding.EncodeToString([]byte("0"))
	}

	value := binary.BigEndian.Uint32(buf)
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(value), 10)))
}

func generateWeChatIlinkClientId() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("casibase-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("casibase-%x", buf)
}
