package certspotter

import (
	"OneLong/Utils"
	outputfile "OneLong/Utils/OutPutfile"
	"OneLong/Utils/gologger"
	"crypto/tls"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"net/http"
	//"strconv"
	//"strings"
	"time"
)

func GetEnInfo(response string, DomainsIP *outputfile.DomainsIP) (*Utils.EnInfos, map[string]*outputfile.ENSMap) {
	//respons := gjson.Get(response, "events").Array()
	//zuo := strings.ReplaceAll(response, "[", "")
	//you := strings.ReplaceAll(zuo, "[", "")
	respons := gjson.Parse(response).Array()
	ensInfos := &Utils.EnInfos{}
	ensInfos.Infos = make(map[string][]gjson.Result)
	ensInfos.SType = "Certspotter"
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
		//ResponseJia := "{" + "\"hostname\"" + ":" + "\"" + respons[aa].String() + "\"" + "}"
		//respons[aa].Raw
		dns_namesArray := gjson.Get(respons[aa].Raw, "dns_names").Array()

		for bb, _ := range dns_namesArray {
			ResponseJia := "{" + "\"dns_names\"" + ":" + "\"" + dns_namesArray[bb].String() + "\"" + "}"
			url := gjson.Parse(ResponseJia).Get("dns_names").String()

			// 检查是否已存在相同的 URL
			if !addedURLs[url] {
				// 如果不存在重复则将 URL 添加到 Infos["Urls"] 中，并在 map 中标记为已添加
				ensInfos.Infos["Urls"] = append(ensInfos.Infos["Urls"], gjson.Parse(ResponseJia))
				addedURLs[url] = true
			}

		}

	}
	//zuo := strings.ReplaceAll(response, "[", "")
	//you := strings.ReplaceAll(zuo, "]", "")

	//ensInfos.Infos["hostname"] = append(ensInfos.Infos["hostname"], gjson.Parse(Result[1].String()))
	//getCompanyInfoById(pid, 1, true, "", options.GetField, ensInfos, options)
	return ensInfos, ensOutMap

}

func Certspotter(domain string, options *Utils.ENOptions, DomainsIP *outputfile.DomainsIP) string {
	gologger.Infof("Certspotter API 查询域名 \n")
	//gologger.Labelf("只实现普通Api 如果是企业修改Api接口 免费的每月250次\n")
	urls := "https://api.certspotter.com/v1/issuances?domain=" + domain + "&include_subdomains=true&expand=dns_names"
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
	time.Sleep(1 * time.Second)
	//加入随机延迟
	time.Sleep(time.Duration(options.GetDelayRTime()) * time.Second)
	clientR := client.R()

	clientR.URL = urls
	resp, _ := clientR.Send()
	if resp.Size() == 4 {
		gologger.Errorf("Certspotter API 未发现域名\n")
		return ""
	}
	res, ensOutMap := GetEnInfo(string(resp.Body()), DomainsIP)

	outputfile.MergeOutPut(res, ensOutMap, "Certspotter", options)
	//outputfile.OutPutExcelByMergeEnInfo(options)
	//
	//Result := gjson.GetMany(string(resp.Body()), "passive_dns.#.address", "passive_dns.#.hostname")
	//AlienvaultResult[0] = append(AlienvaultResult[0], Result[0].String())
	//AlienvaultResult[1] = append(AlienvaultResult[1], Result[1].String())
	//
	//fmt.Printf(Result[0].String())
	return "Success"
}
