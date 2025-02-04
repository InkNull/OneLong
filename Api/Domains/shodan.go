package Domains

import (
	"OneLong/Utils"
	outputfile "OneLong/Utils/OutPutfile"
	"OneLong/Utils/gologger"
	"crypto/tls"
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
func GetEnInfoShodan(response string, domain string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	responselist := gjson.Get(response, "data").Array()
	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Shodan"
	ensOutMap := make(map[string]*outputfile.ENSMap)
	for k, v := range GetENMap() {
		ensOutMap[k] = &outputfile.ENSMap{Name: v.Name, Field: v.Field, KeyWord: v.KeyWord}
	}

	for _, item := range responselist {
		// 从当前条目获取域名
		responsdomain := item.Get("subdomain").String()
		if responsdomain == "" {
			continue
		}
		responsdomain = responsdomain + "." + domain

		// 获取当前条目的所有 IP 地址
		ips := item.Get("value").String() // 假设每个条目下的 "ip" 是一个数组
		re := regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`)

		// 查找匹配的内容
		matches := re.FindAllStringSubmatch(ips, -1)
		if matches == nil {
			continue
		}
		// 为了构建 JSON 字符串，我们先创建 IP 地址的字符串数组
		var ipStrs string

		// 将 IP 地址数组转换为一个字符串，以逗号分隔
		if len(matches) > 1 {
			for _, bb := range matches {
				ipStrs = ipStrs + bb[0] + " , "
			}
		} else {
			ipStrs = matches[0][0]
		}

		// 构建包含 hostname 和所有 IP 地址的 JSON 字符串
		responseJia := fmt.Sprintf("{\"hostname\": \"%s\", \"address\":\"%s\"}", responsdomain, ipStrs)
		DomainsIP.Domains = append(DomainsIP.Domains, responsdomain)

		for _, aa := range matches {
			DomainsIP.IP = append(DomainsIP.IP, aa[0])
		}

		// 将构建的 JSON 字符串解析为 gjson.Result 并追加到 ensInfos.Infos["Urls"]
		ensInfos.Infos["Urls"] = append(ensInfos.Infos["Urls"], gjson.Parse(responseJia))
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

	Utils.DomainTableShow(keyword, data, "Shodan")

	return ensInfos, ensOutMap

}

func Shodan(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) string {
	//gologger.Infof("Shodan 威胁平台查询\n")

	urls := fmt.Sprintf("https://api.shodan.io/dns/domain/%s?key=%s", domain, options.ENConfig.Cookies.Shodan)
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
		gologger.Errorf("Shodan 威胁平台链接访问失败尝试切换代理\n")
		return ""
	}
	if strings.Contains(string(resp.Body()), "No information available for that domain") {
		//gologger.Labelf("Shodan 威胁平台未发现域名 %s\n", domain)
		return ""
	} else if resp.StatusCode() == 404 {
		//gologger.Labelf("Shodan 威胁平台未发现域名 %s\n", domain)
		return ""
	}
	if resp.StatusCode() == 401 {
		gologger.Labelf("Shodan 威胁平台 Token 不正确\n")
		return ""
	}
	res, ensOutMap := GetEnInfoShodan(string(resp.Body()), domain, DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Shodan", options)

	return "Success"
}
