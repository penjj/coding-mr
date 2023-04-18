package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

type CodingResponse[T any] struct {
	Response T `json:"Response"`
}

type Error struct {
	Code    string `json:"Code"`
	Message string `json:"Message"`
}

type DepotsResponse struct {
	Payload   DepotsPayload `json:"Payload"`
	RequestId string        `json:"RequestId"`
	Error     *Error        `json:"Error,omitempty"`
}

type DepotsPayload struct {
	Depots []Depot `json:"Depots"`
}

type Depot struct {
	ID       int    `json:"Id"`
	HTTPSUrl string `json:"HttpsUrl"`
	Name     string `json:"Name"`
}
type MergeResponse struct {
	MergeInfo MergeInfo `json:"MergeInfo"`
	RequestId string    `json:"RequestId"`
	Error     *Error    `json:"Error,omitempty"`
}

type MergeInfo struct {
	MergeRequestId   int              `json:"MergeRequestId"`
	MergeRequestUrl  string           `json:"MergeRequestUrl"`
	MergeRequestInfo MergeRequestInfo `json:"MergeRequestInfo"`
}

type MergeRequestInfo struct {
	Status       string `json:"Status"`
	Author       Author `json:"Author"`
	TargetBranch string `json:"TargetBranch"`
	SourceBranch string `json:"SourceBranch"`
	Title        string `json:"Title"`
}

type Author struct {
	Name string `json:"Name"`
}

type MergeReq struct {
	Action     string
	DepotId    int
	Title      string
	Content    string
	SrcBranch  string
	DestBranch string
}

const CODING_TOKEN_KEY = "user.codingToken"
const WE_ROBOT_URL_KEY = "user.weRobot"

func main() {

	srcBranch, destBranchesStr, content, title := getFlags()
	if len(srcBranch) == 0 {
		srcBranch = getCurrentBranch()
	}
	token := getGitGlobalConfig(CODING_TOKEN_KEY)
	weRobotUrl := getGitGlobalConfig(WE_ROBOT_URL_KEY)
	remoteUrl := getRemoteUrl()
	apiUrl := getApiUrl(remoteUrl)

	userDepots := getUserDepots(apiUrl, token)
	currentDepot := getCurrentDepot(userDepots, remoteUrl)
	destBranches := strings.Split(destBranchesStr, ",")
	mergeList := []MergeInfo{}
	for _, destBranch := range destBranches {
		mergeReqData := MergeReq{
			"CreateGitMergeReq",
			currentDepot.ID,
			title,
			content,
			srcBranch,
			destBranch,
		}
		mergeInfo := sendMergeRequest(apiUrl, token, mergeReqData)
		mergeList = append(mergeList, mergeInfo)
	}
	callWeRobot(weRobotUrl, *currentDepot, mergeList)
}

// 发送消息给企微机器人
func callWeRobot(url string, depot Depot, mergeList []MergeInfo) {
	content := fmt.Sprintf(
		"## 代码合并请求\n\t`%s`在`%s`中发起代码合并请求",
		mergeList[0].MergeRequestInfo.Author.Name,
		depot.Name,
	)

	for _, item := range mergeList {
		content += fmt.Sprintf(
			"\n\t从`%s`合并到%s\n### 合并内容\n\t %s [合并详情](%s)",
			item.MergeRequestInfo.SourceBranch,
			item.MergeRequestInfo.TargetBranch,
			item.MergeRequestInfo.Title,
			item.MergeRequestUrl,
		)
	}
	sendMsg := fmt.Sprintf("{\"msgtype\": \"markdown\",\"markdown\": {\"content\": \"%s\"}}", content)
	fmt.Println(sendMsg)
	// Query depots to find depot ID
	req, err := http.NewRequest("POST", url, strings.NewReader(sendMsg))
	if err != nil {
		color.Red("无法发起网络请求, Error: %s\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		color.Red("企微机器人请求调用失败, Error: %s", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
}

// 当用户还没有设置token的时候，告诉用户如何去设置token
func printHowToSetToken() {
	color.Red("无法获取您的coding authorized token, 请按已下方式进行设置")
	color.White("\te.g.")
	color.Yellow("\t\tgit config --global %s <Your token>", CODING_TOKEN_KEY)
	fmt.Println("")
	color.White("\t如何获取您的coding authorized token? @see:")
	color.Blue("\t\thttps://coding.net/help/docs/member/tokens.html")
}

func getMergeStatus(status string) string {
	statusMap := map[string]string{
		"CANMERGE":       "状态可自动合并",
		"ACCEPTED":       "状态已接受",
		"CANNOTMERGE":    "状态不可自动合并",
		"REFUSED":        "状态已拒绝(关闭)",
		"CANCEL":         "取消",
		"MERGING":        "正在合并中",
		"ABNORMAL":       "状态异常",
		"REVIEW_WAITING": "待评审",
	}

	if desc, ok := statusMap[status]; ok {
		return desc
	}
	return fmt.Sprintf("未知状态: %s", status)
}

// 获取当前分支名
func getCurrentBranch() string {
	// Get current branch
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		color.Red("无法获取当前分支，请检查当前目录下是否存在git仓库 或 当前目录下是否创建分支")
		os.Exit(1)
	}
	return strings.TrimSpace(string(out))
}

// 解析用户输入的参数
// -s 要合并的分支，不传默认区当前分支
// -d 要合并到的分支，需要合并多个分支，可用逗号分隔
// -t 合并标题
// -c 合并内容
func getFlags() (string, string, string, string) {
	// Parse command line arguments
	srcBranch := flag.String("s", "", "Source branch 需要合并的代码分支，默认为当前分支(可选)")
	destBranchesStr := flag.String("d", "", "Dest 需要合并到的分支，多个可用逗号分隔")
	content := flag.String("c", " ", "Content 合并请求内容的详细描述(可选)")
	title := flag.String("t", "", "Title 合并请求标题")
	flag.Parse()
	return *srcBranch, *destBranchesStr, *content, *title
}

// 这个工具是通过coding 的openapi来进行合并操作的, 所以需要用coding 的 token来进行鉴权
// token 需要从openapi中生成，并通过命令设置到环境中
func getGitGlobalConfig(configKey string) string {
	// Get Coding access token
	out, err := exec.Command("git", "config", configKey).Output()
	if err != nil {
		printHowToSetToken()
		os.Exit(1)
	}
	return strings.TrimSpace(string(out))
}

// 执行命令来获取当前仓库的remote url
func getRemoteUrl() string {
	// Get remote URL
	out, err := exec.Command("git", "remote", "get-url", "--all", "origin").Output()
	if err != nil {
		color.Red("无法获取remote url，请检查当前目录下是否有git仓库")
		os.Exit(1)
	}
	return strings.TrimSpace(string(out))
}

// 解析remote url, 获取 team 来拼接 coding openapi 接口地址
func getApiUrl(remote string) string {
	re := regexp.MustCompile(`https://e.coding.net/(\w+)/`)
	search := re.FindStringSubmatch(remote)
	if len(search) <= 0 {
		color.Red("当前仓库不是一个coding仓库： %s", remote)
		os.Exit(1)
	}
	team := search[1]
	apiUrl := fmt.Sprintf("https://%s.coding.net/open-api", team)
	return apiUrl
}

// 查询用户所拥用的仓库列表，用来和当前 remote url对比，来获取 仓库ID 用来发起合并请求
func getUserDepots(apiUrl string, token string) []Depot {

	// Query depots to find depot ID
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader("{\"Action\": \"DescribeMyDepots\"}"))
	if err != nil {
		color.Red("无法发起网络请求, Error: %s\n", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		color.Red("查询用户仓库列表错误, Error: %s", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		color.Red("Coding响应错误%s\n", err)
		os.Exit(1)
	}
	var depotsResp CodingResponse[DepotsResponse]
	err = json.Unmarshal(body, &depotsResp)
	if err != nil {
		color.Red("解析Coding response错误%s\n", err)
		os.Exit(1)
	}
	if depotsResp.Response.Error != nil {
		color.Red("CODING ERROR: %s\n", depotsResp.Response.Error.Message)
	}
	return depotsResp.Response.Payload.Depots
}

// 获取当前仓库的ID
func getCurrentDepot(depots []Depot, remoteUrl string) *Depot {
	var current *Depot
	for _, depot := range depots {
		if depot.HTTPSUrl == remoteUrl {
			current = &depot
			return current
		}
	}
	if current == nil {
		color.Red("您没有当前仓库权限或您修改remote url %s", remoteUrl)
	}
	return current
}

// 发起合并请求
func sendMergeRequest(apiUrl string, token string, mergeReqData MergeReq) MergeInfo {
	mergeReqJson, err := json.Marshal(mergeReqData)
	if err != nil {
		color.Red("当前请求参数无法序列化 %s, ERROR: %s", mergeReqData, err)
		os.Exit(1)
	}
	fmt.Println(string(mergeReqJson))
	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(string(mergeReqJson)))
	if err != nil {
		color.Red("无法发起网络请求 ERROR: %s", err)
		os.Exit(1)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		color.Red("无法发起合并请求 %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		color.Red("无法读取合并请求接口响应内容 %s\n", err)
		os.Exit(1)
	}
	var mergeReqResp CodingResponse[MergeResponse]
	err = json.Unmarshal(body, &mergeReqResp)
	if err != nil {
		color.Red("无法将合并请求响应体json序列化 %s %s\n", string(body), err)
		os.Exit(1)
	}
	response := mergeReqResp.Response
	mergeInfo := response.MergeInfo
	if response.Error != nil {
		color.Red("合并请求发生错误 %s: %s\n", response.Error.Code, response.Error.Message)
		os.Exit(1)
	}
	statusText := getMergeStatus(mergeInfo.MergeRequestInfo.Status)
	color.Blue(
		"已成功发起合并请求 %s => %s\nMergeId: %s\n状态：%s\n",
		mergeReqData.SrcBranch,
		mergeReqData.DestBranch,
		mergeInfo.MergeRequestId,
		statusText,
	)
	color.Green("合并详情地址：%s", mergeInfo.MergeRequestUrl)
	return mergeInfo
}
