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
func GetEnInfoIP138(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	respons := gjson.Get(response, "passive_dns").Array()

	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "IP138	"
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

	Utils.DomainTableShow(keyword, data, "IP138")

	return ensInfos, ensOutMap

}

func IP138(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) string {
	//gologger.Infof("Robtex Api查詢\n")

	IP := fmt.Sprintf("https://site.ip138.com/%s", domain)
	doma := fmt.Sprintf("https://site.ip138.com/%s/domain.htm", domain)
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

	clientR.URL = IP
	respip, err := clientR.Get(IP) //ratelimited

	for add := 1; add < 4; add += 1 {
		if respip.RawResponse == nil {
			respip, _ = clientR.Get(IP)
			time.Sleep(3 * time.Second)
		} else if respip.Body() != nil {
			break
		}
	}

	if err != nil {
		gologger.Errorf("Crtsh API 链接访问失败尝试切换代理\n")
		return ""
	}
	if strings.Contains(string(respip.Body()), "禁止查询该域名") {
		gologger.Labelf("IP138禁止查询该域名 %s\n", domain)
		return ""
	}
	clientdom := client.R()
	respdomain, _ := clientdom.Get(doma)
	if strings.Contains(string(respdomain.Body()), "未查找到结果！") && strings.Contains(string(respip.Body()), "未查找到结果！") {
		//gologger.Labelf("IP138未发现域名 %s\n", domain)
		return ""
	}

	var hostname []string
	var address []string

	rehostname := regexp.MustCompile(`(?:[a-z0-9](?:[a-z0-9\-]{0,61}[a-z0-9])?\.)+` + regexp.QuoteMeta(domain))
	reip := regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`)

	ipbuffer := reip.FindAllStringSubmatch(strings.TrimSpace(string(respip.Body())), -1)
	hostnamebuffer := rehostname.FindAllStringSubmatch(strings.TrimSpace(string(respdomain.Body())), -1)
	if ipbuffer != nil {
		for _, aa := range ipbuffer {
			address = append(address, aa[0])
		}

	}
	if hostnamebuffer != nil {
		for _, aa := range hostnamebuffer {
			hostname = append(hostname, aa[0])
		}
	}
	address = Utils.SetStr(address)
	hostname = Utils.SetStr(hostname)
	if len(address) == 0 || len(hostname) == 0 {
		return ""
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
	res, ensOutMap := GetEnInfoIP138(result1, DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "IP138", options)

	return "Success"
}
