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
	//"strconv"
	//"strings"
	"time"
)

// 用于保护 addedURLs
func GetEnInfoWhoisxmlapi(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	responselist := gjson.Get(response, "result.records.#.domain").Array()
	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Whoisxmlapi"
	ensOutMap := make(map[string]*outputfile.ENSMap)
	for k, v := range GetENMap() {
		ensOutMap[k] = &outputfile.ENSMap{Name: v.Name, Field: v.Field, KeyWord: v.KeyWord}
	}

	addedURLs := make(map[string]bool)
	for aa, _ := range responselist {
		ResponseJia := "{" + "\"hostname\"" + ":" + "\"" + responselist[aa].String() + "\"" + "}"
		url := gjson.Parse(ResponseJia).Get("hostname").String()

		// 检查是否已存在相同的 URL
		if !addedURLs[url] {
			DomainsIP.Domains = append(DomainsIP.Domains, url)
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

	Utils.DomainTableShow(keyword, data, "Whoisxmlapi")

	return ensInfos, ensOutMap

}

func Whoisxmlapi(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) string {
	//gologger.Infof("Whoisxmlapi Api查询\n")

	urls := fmt.Sprintf("https://subdomains.whoisxmlapi.com/api/v1?apiKey=%s&domainName=%s", options.ENConfig.Cookies.Whoisxmlapi, domain)
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
		gologger.Errorf("Whoisxmlapi API 链接访问失败尝试切换代理\n")
		return ""
	}
	if resp.Body() == nil || resp.StatusCode() == 422 {
		//gologger.Labelf("Whoisxmlapi Api查询未发现域名 %s\n", domain)
		return ""
	} else if resp.StatusCode() == 403 {
		gologger.Errorf("Whoisxmlapi Api Cookie失效 %s\n", domain)
		return ""
	}
	res, ensOutMap := GetEnInfoWhoisxmlapi(string(resp.Body()), DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Whoisxmlapi", options)

	return "Success"
}
