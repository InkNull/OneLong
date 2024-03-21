package Baidu

import (
	"OneLong/Utils"
	outputfile "OneLong/Utils/OutPutfile"
	"OneLong/Utils/gologger"
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	//"strconv"
	//"strings"
	"time"
)

var wg sync.WaitGroup

func GetEnInfo(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	respons := gjson.Get(response, "passive_dns").Array()
	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Baidu"
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

func AlienvaultLogin(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) {
	//gologger.Infof("Alienvault\n")
	//urls := "https://otx.alienvault.com/api/v1/indicators/domain/" + domain + "/url_list?limit=100&page=11"
	//addedURLs := make(map[string]bool)
	//wds := []string{"inurl:admin", "inurl:login", "inurl:system", "inurl:register", "后台", "系统", "登录", "管理", "平台"}
	dir := filepath.Join(Utils.GetPathDir(), "Script/Dict/Login.txt")
	file, err := os.Open(dir)
	if err != nil {
		gologger.Errorf("无法打开文件后台目录文件%s\n", dir)
		return
	}
	defer file.Close()

	// 使用哈希集合存储文本中的内容
	contentSet := make(map[string]bool)

	// 创建 Scanner 对象
	scanner := bufio.NewScanner(file)

	// 逐行读取文件内容
	for scanner.Scan() {
		line := scanner.Text()
		contentSet[line] = true
	}
	urls := fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/url_list?limit=100&page=1", domain)
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
		gologger.Errorf("Alienvault API 链接访问失败尝试切换代理\n")
		return
	}

	count := gjson.GetBytes(resp.Body(), "full_size").Int()
	if count == 0 {
		gologger.Labelf("Alienvault 未发现后台域名 %s\n", domain)
		return
	}
	intcount := int(count)
	for add := 1; add <= intcount/100+1; add += 1 {
		urls = fmt.Sprintf("https://otx.alienvault.com/api/v1/indicators/domain/%s/url_list?limit=100&page=%d", domain, add)
		resp, err = clientR.Get(urls)
		loginurls := gjson.GetBytes(resp.Body(), "url_list.#.url").Array()

		for _, loginurl := range loginurls {
			wg.Add(1)
			loginurl := loginurl
			go func() {
				for content := range contentSet {
					if strings.Contains(loginurl.String(), content) {
						fmt.Println("匹配到链接:", loginurl.String())
						DomainsIP.LoginUrl = append(DomainsIP.LoginUrl, loginurl.String())
					}
				}
				wg.Done()
			}()

		}
		wg.Wait()

	}

}
