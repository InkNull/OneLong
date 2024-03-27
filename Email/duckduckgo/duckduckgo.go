package duckduckgo

import (
	"OneLong/Utils"
	outputfile "OneLong/Utils/OutPutfile"
	"OneLong/Utils/gologger"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"net/http"
	"regexp"
	"strings"
	"sync"

	//"strconv"
	//"strings"
	"time"
)

func GetEnInfo(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	respons := gjson.Get(response, "passive_dns").Array()
	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "alienvault"
	ensOutMap := make(map[string]*outputfile.ENSMap)
	for k, v := range getENMap() {
		ensOutMap[k] = &outputfile.ENSMap{Name: v.name, Field: v.field, KeyWord: v.keyWord}
	}
	//Result := gjson.GetMany(response, "passive_dns.#.address", "passive_dns.#.hostname")
	//ensInfos.Infoss = make(map[string][]map[string]string)
	//获取公司信息
	//ensInfos.Infos["passive_dns"] = append(ensInfos.Infos["passive_dns"], gjson.Parse(Result[0].String()))
	addedURLs := make(map[string]bool)
	for aa, _ := range respons {
		if strings.Contains(respons[aa].String(), "address") {
			re := regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`)
			ip := gjson.Get(respons[aa].String(), "address").String()
			matches := re.FindAllStringSubmatch(strings.TrimSpace(ip), -1)
			for _, bu := range matches {
				if !addedURLs[bu[0]] {
					// 如果不存在重复则将 URL 添加到 Infos["Urls"] 中，并在 map 中标记为已添加
					DomainsIP.IP = append(DomainsIP.IP, bu[0])
					addedURLs[bu[0]] = true
				}
				break
			}

		}
		if strings.Contains(respons[aa].String(), "hostname") {
			hostname := gjson.Get(respons[aa].String(), "hostname").String()
			if !addedURLs[hostname] {
				// 如果不存在重复则将 URL 添加到 Infos["Urls"] 中，并在 map 中标记为已添加
				DomainsIP.Domains = append(DomainsIP.Domains, hostname)
				addedURLs[hostname] = true
			}

		}
		ensInfos.Infos["Urls"] = append(ensInfos.Infos["Urls"], gjson.Parse(respons[aa].String()))
	}

	//zuo := strings.ReplaceAll(response, "[", "")
	//you := strings.ReplaceAll(zuo, "]", "")

	//ensInfos.Infos["hostname"] = append(ensInfos.Infos["hostname"], gjson.Parse(Result[1].String()))
	//getCompanyInfoById(pid, 1, true, "", options.Getfield, ensInfos, options)
	return ensInfos, ensOutMap

}

func ParseUrl(text string) []string {
	var urls []string
	mapurls := make(map[string]bool)

	var load map[string]interface{}
	err := json.Unmarshal([]byte(text), &load)
	if err != nil {
		fmt.Println("解析JSON失败:", err)
	}

	for _, val := range load {
		switch v := val.(type) {
		case int, nil:
			continue
		case map[string]interface{}:
			for _, value := range v {
				if str, ok := value.(string); ok && str != "" && (strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")) {
					urls = append(urls, str)
					mapurls[str] = true
				}
			}
		case []interface{}:
			if len(v) == 0 {
				continue
			}
			if innerMap, ok := v[0].(map[string]interface{}); ok {
				for _, value := range innerMap {
					if str, ok := value.(string); ok && str != "" && (strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")) {
						urls = append(urls, str)
						mapurls[str] = true

					}
				}
			}
		case string:
			if v != "" && (strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://")) {
				urls = append(urls, v)
				mapurls[v] = true
			}
		default:
			continue
		}
	}

	return urls

}

func clearresponse(results string) string {

	replacements := []string{
		"<em>", "</em>", // 替换 <em> 和 </em>
		"<b>", "</b>", // 替换 <b> 和 </b>
		"%3a",                   // 替换 %3a
		"<strong>", "</strong>", // 替换 <strong> 和 </strong>
		"<wbr>", "</wbr>", // 替换 <wbr> 和 </wbr>
	}
	replacements2 := []string{
		"<", ">", ":", "=", ";", "&", "%3A", "%3D", "%3C", "%2f", "/", "\\", // 其他需要替换的字符
	}

	// 执行替换
	for _, search := range replacements {
		results = strings.ReplaceAll(results, search, "")
	}
	for _, search := range replacements2 {
		results = strings.ReplaceAll(results, search, " ")
	}
	return results

}

func parseurl(domain string, options *Utils.ENOptions) string {

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

	client.Header.Set("Content-Type", "application/json")
	client.Header.Del("Cookie")

	//强制延时1s
	time.Sleep(1 * time.Second)
	//加入随机延迟
	time.Sleep(time.Duration(options.GetDelayRTime()) * time.Second)
	clientR := client.R()

	clientR.URL = domain
	resp, err := clientR.Get(domain)

	for add := 1; add < 4; add += 1 {
		if resp.RawResponse == nil {
			resp, _ = clientR.Get(domain)
			time.Sleep(1 * time.Second)
		} else if resp.Body() != nil {
			break
		}
	}

	if err != nil {
		gologger.Errorf("%s链接访问失败尝试切换代理\n", domain)
	}
	return string(resp.Body())
}

func Duckduckgo(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) {
	//gologger.Infof("Alienvault\n")
	var wg sync.WaitGroup
	var respnsehe string
	//domain = "https://api.duckduckgo.com/?q=nthu.edu.tw&format=json&pretty=1"
	urls := "https://api.duckduckgo.com/?q=" + domain + "&format=json&pretty=1"

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

	client.Header.Set("Content-Type", "application/json")
	client.Header.Del("Cookie")

	//强制延时1s
	time.Sleep(1 * time.Second)
	//加入随机延迟
	time.Sleep(time.Duration(options.GetDelayRTime()) * time.Second)
	clientR := client.R()

	clientR.URL = urls
	resp, err := clientR.Get(urls)

	for add := 1; add < 4; add += 1 {
		if resp.RawResponse == nil {
			resp, _ = clientR.Get(urls)
			time.Sleep(1 * time.Second)
		} else if resp.Body() != nil {
			break
		}
	}

	if err != nil {
		gologger.Errorf("Yahoo 链接访问失败尝试切换代理\n")

	}
	parse := Utils.SetStr(ParseUrl(string(resp.Body())))

	for _, url := range parse {
		url := url
		wg.Add(1)
		go func() {
			respnsehe += parseurl(url, options)
			wg.Done()
		}()

	}
	wg.Wait()
	respnsehe = clearresponse(respnsehe)
	Email := `[a-zA-Z0-9.\-_+#~!$&',;=:]+@` + `[a-zA-Z0-9.-]*` + strings.ReplaceAll(domain, "www.", "")

	re := regexp.MustCompile(Email)

	matches := re.FindAllStringSubmatch(strings.TrimSpace(respnsehe), -1)
	for _, aa := range matches {

		fmt.Print(aa)
	}
	//res, ensOutMap := GetEnInfo(string(resp.Body()), DomainsIP)
	//
	//outputfile.MergeOutPut(res, ensOutMap, "alienvault", options)
	//

}
