package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Respon struct {
	// {"errcode":0,"errmsg":"ok"}
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// Hook Grafana webhook 结构体
// https://grafana.com/docs/grafana/v8.4/alerting/unified-alerting/contact-points/#alert
type Hook struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	RrgId             int               `json:"orgId"`
	Alerts            []Alert           `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
	Title             string            `json:"title"`
	State             string            `json:"state"`
	Message           string            `json:"message"`
	RuleURL           string            `json:"ruleUrl"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	ValueString  string            `json:"valueString"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
	SilenceURL   string            `json:"silenceURL"`
	DashboardURL string            `json:"dashboardURL"`
	PanelURL     string            `json:"panelURL"`
}

var sentCount = 0
var startTime = time.Now().Format(DateFormat)

const (
	Url         = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key="
	OKMsg       = "告警恢复"
	AlertingMsg = "触发告警"
	OK          = "OK"
	Alerting    = "Alerting"
	ColorGreen  = "info"
	ColorGray   = "comment"
	ColorRed    = "warning"
	DateFormat  = "2006-01-02"
)

// GetSendCount 记录发送次数
func GetSendCount(c *gin.Context) {
	_, _ = c.Writer.WriteString("G2WW Server created by Nova Kwok is running! Parsed & forwarded " + strconv.Itoa(sentCount) + " messages to WeChat Work!")
	return
}

// SendMsg 发送消息
func SendMsg(c *gin.Context) {
	h := &Hook{}
	if err := c.BindJSON(&h); err != nil {
		fmt.Println(err)
		_, _ = c.Writer.WriteString("Error on JSON format")
		return
	}
	// 每日报警次数清零
	currentTime := time.Now()
	if currentTime.Add(-time.Hour*24).Format(DateFormat) == startTime {
		startTime = currentTime.Format(DateFormat)
		sentCount = 0
	}
	marshal, _ := json.Marshal(h)
	fmt.Println("接受参数数据：", string(marshal))
	sentCount++
	color := ColorGreen
	if !strings.Contains(h.Title, OK) {
		color = ColorRed
	}
	msg := fmt.Sprintf(`<font color=\"%s\">今日报警: %d 次, 本次报警: %d 条</font>\r\n`, color, sentCount, len(h.Alerts))
	// 封装报警内容, 提取 Labels 中的 alertname 和 Annotations 中的 summary
	for _, v := range h.Alerts {
		msg = msg + fmt.Sprintf(`<font color=\"%s\">%s</font>\r\n<font color=\"comment\">%s\r\n</font>`, color, v.Labels["alertname"], v.Annotations["summary"])
	}
	// TODO
	// {"errcode":40058,"errmsg":"markdown.content exceed max length 4096. invalid Request Parameter, hint: [1672133087235733136500908], from ip: more info at https://open.work.weixin.qq.com/devtool/query?e=40058"}
	// webchat 不允许超过 4096 字节,这个应该怎么样处理呢？
	if len(msg) > 4096 {
		msg = `<font color=\"warning\">err: content exceed max length 4096</font>`
		if sentCount >= 1 {
			sentCount--
		}
		marshal, _ := json.Marshal(h)
		fmt.Println("接收参数数据(大于4096个字节)：", string(marshal))
	}
	// Send to WeChat Work
	url := Url + c.Query("key")
	// 处理数据格式
	msgStr := MsgMarkdown(msg)

	fmt.Println("发送的消息是：", msgStr)

	// 发送http请求
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(msgStr)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.Writer.WriteString("Error sending to WeChat Work API") // nolint
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("微信返回Body: ", string(body))
	respon := Respon{}
	if err := json.Unmarshal(body, &respon); err != nil {
		c.Writer.WriteString(fmt.Sprintf("json unmarshal error: %s\n", string(body)))
		return
	}
	if respon.Errcode != 0 {
		fmt.Println("发送的消息是：", msgStr)
		fmt.Println("返回结果:", respon.Errmsg)
	}

	_, _ = c.Writer.Write(body)

	return
}

// MsgMarkdown 企业微信 markdown 格式
// https://developer.work.weixin.qq.com/document/path/91770#markdown%E7%B1%BB%E5%9E%8B
// 发送消息类型
func MsgMarkdown(content string) string {
	return fmt.Sprintf(`
        {
       "msgtype": "markdown",
       "markdown": {
           "content": "%s"
       }
  }`, content)
}
