package Domains

import (
	"OneLong/Utils"
	outputfile "OneLong/Utils/OutPutfile"
	"OneLong/Utils/gologger"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"net/http"
	"regexp"
	"strings"
	//"strconv"
	//"strings"
	"time"
)

// 用于保护 addedURLs
func GetEnInfoFofa(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	respons := gjson.Get(response, "results").Array()

	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Fofa"
	ensOutMap := make(map[string]*outputfile.ENSMap)
	for k, v := range GetENMap() {
		ensOutMap[k] = &outputfile.ENSMap{Name: v.Name, Field: v.Field, KeyWord: v.KeyWord}
	}

	addedURLs := make(map[string]bool)
	hostname := `(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`
	// 编译正则表达式
	re := regexp.MustCompile(hostname)

	// 查找匹配的内容

	for aa, _ := range respons {
		Fen := strings.SplitN(respons[aa].String(), ",", 2)
		Fen[1] = strings.ReplaceAll(Fen[1], "\"", "")
		Fen[1] = strings.ReplaceAll(Fen[1], "]", "")
		matches := re.FindAllStringSubmatch(respons[aa].String(), -1)

		ResponseJia := fmt.Sprintf("{\"hostname\": \"%s\", \"address\": \"%s\"}", matches[0][0], Fen[1])

		url := gjson.Parse(ResponseJia).Get("hostname").String()
		ip := gjson.Parse(ResponseJia).Get("address").String()
		DomainsIP.Domains = append(DomainsIP.Domains, url)
		DomainsIP.IP = append(DomainsIP.IP, ip)
		// 检查是否已存在相同的 URL
		if !addedURLs[url] {
			// 如果不存在重复则将 URL 添加到 Infos["Urls"] 中，并在 map 中标记为已添加
			ensInfos.Infos["Urls"] = append(ensInfos.Infos["Urls"], gjson.Parse(ResponseJia))
			addedURLs[url] = true
		}

	}

	//命令输出展示

	var data [][]string
	var keyword []string
	for _, y := range GetENMap() {
		for _, ss := range y.KeyWord {
			if ss == "数据关联" {
				continue
			}
			keyword = append(keyword, ss)
		}

		for _, res := range ensInfos.Infos["Urls"] {
			results := gjson.GetMany(res.Raw, y.Field...)
			var str []string
			for _, s := range results {
				str = append(str, s.String())
			}
			data = append(data, str)
		}

	}

	Utils.DomainTableShow(keyword, data, "Fofa")

	return ensInfos, ensOutMap

}

func Fofa(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) string {

	qbase64 := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("domain=\"%s\"", domain)))
	urls := fmt.Sprintf("https://fofa.info/api/v1/search/all?full=true&fields=host,ip&page=1&size=1000&email=%s&key=%s&qbase64=%s", options.ENConfig.Cookies.FofaEmail, options.ENConfig.Cookies.FofaKey, qbase64)
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	client.SetTimeout(time.Duration(options.TimeOut) * time.Minute)
	if options.Proxy != "" {
		client.SetProxy(options.Proxy)
	}
	client.Header = http.Header{
		"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"},
		"Accept":     {"text/html,application/json,application/xhtml+xml, image/jxr, */*"},
	}

	client.Header.Del("Cookie")

	//强制延时1s
	time.Sleep(3 * time.Second)
	//加入随机延迟
	time.Sleep(time.Duration(options.GetDelayRTime()) * time.Second)
	clientR := client.R()

	clientR.URL = urls
	resp, err := clientR.Get(urls)
	for add := 1; add < 4; add += 1 {
		if resp.RawResponse == nil {
			resp, _ = clientR.Get(urls)
			time.Sleep(3 * time.Second)
		} else if resp.Body() != nil {
			break
		}
	}
	if err != nil {
		gologger.Errorf("Fofa 威胁平台链接访问失败尝试切换代理\n")
		return ""
	}
	if gjson.Get(string(resp.Body()), "size").Int() == 0 {

		return ""
	}
	res, ensOutMap := GetEnInfoFofa(string(resp.Body()), DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Fofa", options)

	return "Success"
}
