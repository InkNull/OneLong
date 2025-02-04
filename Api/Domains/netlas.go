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
	"net/url"
	"strconv"
	"strings"
	//"strconv"
	//"strings"
	"time"
)

// 用于保护 addedURLs
func GetEnInfoNetlas(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	respons := gjson.Get(response, "passive_dns").Array()

	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Netlas"
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

	Utils.DomainTableShow(keyword, data, "Netlas")

	return ensInfos, ensOutMap

}

func Netlas(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) string {

	//gologger.Infof("Netlas 威胁平台查询\n")
	//urls := "https://leakix.net/api/subdomains/" + domain
	endpoint := "https://app.netlas.io/api/domains_count/"
	params := url.Values{}
	countQuery := fmt.Sprintf("domain:*.%s AND NOT domain:%s", domain, domain)
	params.Set("q", countQuery)
	urls := endpoint + "?" + params.Encode()

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	client.SetTimeout(time.Duration(options.TimeOut) * time.Minute)
	if options.Proxy != "" {
		client.SetProxy(options.Proxy)
	}
	client.Header = http.Header{
		"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"},
		"Accept":     {"text/html,application/json,application/xhtml+xml, image/jxr, */*"},
		"api-key":    {options.ENConfig.Cookies.Netlas},
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
		gologger.Errorf("Netlas 威胁平台访问失败尝试切换代理\n")
		return ""
	}
	if gjson.Get(string(resp.Body()), "count").Int() == 0 {
		//gologger.Labelf("Netlas 威胁平台未发现域名 %s\n", domain)
		return ""
	}
	if strings.Contains(string(resp.Body()), "You can wait while daily rate limit will ") {
		gologger.Errorf("请切换IP 当日访问上限")
	}
	count := gjson.Get(string(resp.Body()), "count").Int()
	var address []string
	var hostname []string
	var ipss string
	for i := 0; i < int(count); i += 20 {
		endpoint := "https://app.netlas.io/api/domains/"
		params := url.Values{}
		offset := strconv.Itoa(i)
		query := fmt.Sprintf("domain:(domain:*.%s AND NOT domain:%s)", domain, domain)
		params.Set("q", query)
		params.Set("source_type", "include")
		params.Set("start", offset)
		params.Set("fields", "*")
		apiUrl := endpoint + "?" + params.Encode()
		client.Header = http.Header{
			"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"},
			"Accept":     {"text/html,application/json,application/xhtml+xml, image/jxr, */*"},
			"api-key":    {"R5yHhXQgud0eDV34IR8TUck3AchS99dS"},
		}
		clientR := client.R()

		clientR.URL = apiUrl
		resp, _ := clientR.Get(urls)
		buff := gjson.Get(string(resp.Body()), "items.#.data").Array()
		for _, item := range buff {
			ips := item.Get("a").Array()
			hostnames := item.Get("domain").String()
			for _, ip := range ips {
				ipss = ipss + ip.String() + "\n"
			}
			address = append(address, ipss)
			hostname = append(hostname, hostnames)
			ipss = ""

		}
	}
	if len(hostname) == 0 || len(address) == 0 {
		return ""
	}
	passive_dns := "{\"passive_dns\":["
	var add int
	for add = 0; add < len(hostname); add++ {
		passive_dns += "{\"hostname\"" + ":" + "\"" + hostname[add] + "\"" + "," + "\"address\"" + ":" + "\"" + address[add] + "\"" + "},"
		ips := strings.Split(address[add], "\n")
		for _, ip := range ips {
			DomainsIP.IP = append(DomainsIP.IP, ip)
		}
		DomainsIP.Domains = append(DomainsIP.Domains, hostname[add])

	}
	passive_dns = passive_dns + "]}"
	res, ensOutMap := GetEnInfoNetlas(passive_dns, DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Netlas", options)

	return "Success"
}
