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
func GetEnInfoRobtex(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	respons := gjson.Get(response, "passive_dns").Array()

	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Robtex"
	ensOutMap := make(map[string]*outputfile.ENSMap)
	for k, v := range GetENMap() {
		ensOutMap[k] = &outputfile.ENSMap{Name: v.Name, Field: v.Field, KeyWord: v.KeyWord}
	}

	for aa, _ := range respons {
		ensInfos.Infos["Urls"] = append(ensInfos.Infos["Urls"], gjson.Parse(respons[aa].String()))
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

	Utils.DomainTableShow(keyword, data, "Robtex")

	return ensInfos, ensOutMap

}

func Robtex(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) string {
	//gologger.Infof("Robtex Api查詢\n")

	urls := fmt.Sprintf("https://freeapi.robtex.com/pdns/forward/%s", domain)
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	client.SetTimeout(time.Duration(options.TimeOut) * time.Minute)
	if options.Proxy != "" {
		client.SetProxy(options.Proxy)
	}
	client.Header = http.Header{
		"User-Agent":   {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"},
		"Accept":       {"text/html,application/json,application/xhtml+xml, image/jxr, */*"},
		"Content-Type": {"application/x-ndjson"},
	}

	client.Header.Del("Cookie")

	//强制延时1s
	time.Sleep(3 * time.Second)
	//加入随机延迟
	time.Sleep(time.Duration(options.GetDelayRTime()) * time.Second)
	clientR := client.R()

	clientR.URL = urls
	resp, err := clientR.Get(urls) //ratelimited

	for add := 1; add < 4; add += 1 {
		if resp.RawResponse == nil {
			resp, _ = clientR.Get(urls)
			time.Sleep(3 * time.Second)
		} else if resp.Body() != nil {
			break
		}
	}
	if err != nil {
		gologger.Errorf("Crtsh API 链接访问失败尝试切换代理\n")
		return ""
	}
	if len(resp.Body()) == 0 {
		//gologger.Labelf("Robtex Api 未发现域名 %s\n", domain)
		return ""
	}
	if strings.Contains(string(resp.Body()), "ratelimited") {
		gologger.Labelf("请稍后使用Robtex Api查詢 已限速 \n")
		return ""
	}
	var hostname []string
	var address []string
	result := "{\"passive_dns\":[" + string(resp.Body()) + "]}"
	responselist := gjson.Get(result, "passive_dns.#.rrdata").Array()
	for aa, _ := range responselist {
		ip := regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`)
		host := regexp.MustCompile(`(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`)
		if ip.FindAllStringSubmatch(strings.TrimSpace(responselist[aa].String()), -1) != nil {
			address = append(address, responselist[aa].String())
		} else if host.FindAllStringSubmatch(strings.TrimSpace(responselist[aa].String()), -1) != nil {
			hostname = append(hostname, responselist[aa].String())
		}

	}
	var add int
	result1 := "{\"passive_dns\":["
	if len(hostname) < len(address) {
		for add = 0; add < len(hostname); add++ {
			result1 += "{\"hostname\"" + ":" + "\"" + hostname[add] + "\"" + "," + "\"address\"" + ":" + "\"" + address[add] + "\"" + "},"
			DomainsIP.Domains = append(DomainsIP.Domains, hostname[add])
			DomainsIP.IP = append(DomainsIP.IP, address[add])
		}
		for ii := add; ii < len(address); ii++ {
			result1 += "{\"address\"" + ":" + "\"" + address[ii] + "\"" + "},"
			DomainsIP.IP = append(DomainsIP.IP, address[ii])
		}

	} else {
		for add = 0; add < len(address); add++ {
			result1 += "{\"hostname\"" + ":" + "\"" + hostname[add] + "\"" + "," + "\"address\"" + ":" + "\"" + address[add] + "\"" + "},"
			DomainsIP.Domains = append(DomainsIP.Domains, hostname[add])
			DomainsIP.IP = append(DomainsIP.IP, address[add])
		}
		for ii := add; ii < len(hostname); ii++ {
			result1 += "{\"hostname\"" + ":" + "\"" + hostname[ii] + "\"" + "},"
			DomainsIP.Domains = append(DomainsIP.Domains, hostname[ii])
		}
	}

	result1 = result1 + "]}"
	res, ensOutMap := GetEnInfoRobtex(result1, DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Robtex Api", options)

	return "Success"
}
