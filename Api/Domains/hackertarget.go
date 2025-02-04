package Domains

import (
	"OneLong/Utils"
	outputfile "OneLong/Utils/OutPutfile"
	"OneLong/Utils/gologger"
	"crypto/tls"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
	"time"
)

// 用于保护 addedURLs
func GetEnInfoHackertarget(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {

	respons := gjson.Get(response, "passive_dns").Array()
	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Hackertarget"
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

	Utils.DomainTableShow(keyword, data, "Hackertarget")

	return ensInfos, ensOutMap

}

func Hackertarget(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) string {
	//gologger.Infof("Hackertarget API 查询 \n")
	//gologger.Labelf("只实现普通Api 如果是企业修改Api接口 免费的每月250次\n")
	urls := "https://api.hackertarget.com/hostsearch/?q=" + domain
	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	client.SetTimeout(time.Duration(options.TimeOut) * time.Minute)
	if options.Proxy != "" {
		client.SetProxy(options.Proxy)
		//client.SetProxy("192.168.203.111:1111")
	}
	client.Header = http.Header{
		"User-Agent": {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"},
		"Accept":     {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9"},
		//"X-Key":      {options.ENConfig.Cookies.Binaryedge},
	}

	client.Header.Set("Content-Type", "application/json")
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
		gologger.Errorf("Hackertarget API 链接访问失败尝试切换代理\n")
		return ""
	}
	//error invalid host
	if strings.Contains(string(resp.Body()), "error invalid host") {
		gologger.Labelf("Hackertarget API 未发现到域名\n")
		return ""
	} else if strings.Contains(string(resp.Body()), "Increase Quota with Membership") {
		gologger.Errorf("Hackertarget API 次数使用完\n")
		return ""
	}
	lines := strings.Split(string(resp.Body()), "\n")

	// 遍历每一行
	passive_dns := "{\"passive_dns\":["
	for _, line := range lines {
		// 按空格分割键值对
		parts := strings.SplitN(line, ",", 2)
		if len(parts) != 2 {
			break // 如果不是键值对格式，跳过
		}
		passive_dns += "{\"hostname\"" + ":" + "\"" + parts[0] + "\"" + "," + "\"address\"" + ":" + "\"" + parts[1] + "\"" + "},"
		DomainsIP.Domains = append(DomainsIP.Domains, parts[0])
		DomainsIP.IP = append(DomainsIP.IP, parts[1])
	}

	passive_dns = passive_dns + "]}"
	res, ensOutMap := GetEnInfoHackertarget(passive_dns, DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Hackertarget Api", options)

	return "Success"
}
