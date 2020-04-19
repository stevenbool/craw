package craw

/*
用于封装常用的抓取搜索结果的模块
*/
import (
	"compress/gzip"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/parnurzeal/gorequest"
	"github.com/tidwall/gjson"
)

//定义一个全局的错误列表
//只要出错了就加载到列表，返回errs
type errlist []error

//Site recover  site info
type Site struct {
	//排序
	SortID int
	//收录的标题
	Title string
	//收录的描述
	Description string
	//收录的最新时间
	Uptime string
	//收录的真实url
	Reallink string
	//是否是首页
	Isindex string
	//收录的链接
	Showlink string
	//获取来源
	Src string
}

//Craw recover list function for search
type Craw struct{}

//New start craw moudle
func New() *Craw {
	return &Craw{}
}

//BaiduMobileAboutWord 百度移动相关搜索
func (c *Craw) BaiduMobileAboutWord(qey *goquery.Document) []string {
	list := make([]string, 0)
	qey.Find(".rw-list-container span").Each(func(_ int, g *goquery.Selection) {
		list = append(list, g.Text())
	})
	return list
}

//BaiduMobileOtherWord 百度移动其他人相关搜索
func (c *Craw) BaiduMobileOtherWord(qey *goquery.Document) []string {
	list := make([]string, 0)
	qey.Find(".span-item span").Each(func(_ int, g *goquery.Selection) {
		list = append(list, g.Text())
	})
	return list
}

//BaiduMobildAllWord 百度移动端所有拓展关键词
func (c *Craw) BaiduMobildAllWord(qey *goquery.Document) []string {
	list := make([]string, 0)
	list = append(list, c.BaiduMobileAboutWord(qey)...)
	list = append(list, c.BaiduMobileOtherWord(qey)...)
	return list
}

//BaiduMobileSort 获取百度移动端排名 这个功能需要代理
func (c *Craw) BaiduMobileSort(qey *goquery.Document, iplist ...string) []*Site {
	list := make([]*Site, 0)
	qey.Find("#results>.c-result").Each(func(id int, g *goquery.Selection) {
		json := g.AttrOr("data-log", "")
		json = strings.ReplaceAll(json, `'`, `"`)
		if !gjson.Valid(json) {
			log.Println("不是有效的json字符串")
		}
		reallink := gjson.Get(json, "mu").String()

		g = g.Find(".c-result-content")
		title := g.Find("h3").Text()
		uptime := g.Find(".c-line-clamp3>.c-gap-right-small").Text()
		des := g.Find(".c-line-clamp3>span").Last().Text()
		showlink := g.Find(".c-line-clamp1>span").Text()

		isindex := c.IsIndex(reallink)
		list = append(list, &Site{
			SortID:      id + 1,
			Title:       title,
			Uptime:      uptime,
			Description: des,
			Reallink:    reallink,
			Isindex:     isindex,
			Showlink:    showlink,
			Src:         "百度移动",
		})
	})
	return list
}

//BaiduReallink 获取百度移动真实的链接
func (c *Craw) BaiduReallink(link string, iplist ...string) string {
	if link == "" {
		return ""
	}
	client := c.BaiduClient()
	if len(iplist) != 0 {
		//随机获取其中一个IP
		ip := c.RandMemberFromSlice(iplist)
		client = client.Proxy(ip)
	}
	_, _, errs := client.RedirectPolicy(c.NoRedict).Get(link).End()
	if errs == nil {
		log.Println("获取百度真实链接时未获取到有效的错误信息")
		return ""
	}
	reallink, ok := errs[0].(*url.Error)
	if ok {
		log.Println("获取百度真实链接时出现不可意料的错误")
		return ""
	}
	return reallink.URL
}

//BaiduClient 获取一个用于百度的客户端
func (c *Craw) BaiduClient() *gorequest.SuperAgent {
	return gorequest.New().
		Retry(3, 5*time.Second, http.StatusBadRequest, http.StatusInternalServerError).
		AppendHeader("Accept", "*/*").AppendHeader("Accept-Encoding", "gzip, deflate, br")
}

//NoRedict  获取没有重定向的匿名函数
func (c *Craw) NoRedict(req gorequest.Request, via []gorequest.Request) error {
	if len(via) > 0 {
		return errors.New("本次请求不允许重定向")
	}
	return nil
}

//IsIndex 判断是否是首页
func (c *Craw) IsIndex(link string) string {
	if link == "" {
		return "空链接"
	}
	linkinfo, err := url.Parse(link)
	if err != nil {
		return "无效链接"
	}
	if linkinfo.RawQuery == "" && linkinfo.Path == "" || linkinfo.Path == "/" {
		return "首页"
	}
	return "内页"
}

//SmAboutWord 神马相关搜索
func (c *Craw) SmAboutWord(qey *goquery.Document) []string {
	list := make([]string, 0)
	qey.Find(".news-title").Each(func(i int, g *goquery.Selection) {
		list = append(list, g.Text())
	})
	return list
}

//SmOtherWord 神马其他人还搜了
func (c *Craw) SmOtherWord(qey *goquery.Document) []string {
	list := make([]string, 0)
	qey.Find(".c-e-btn-text").Each(func(i int, g *goquery.Selection) {
		list = append(list, g.Text())
	})
	return list
}

//SmAllWord 根据goquery获取所有神马关键词
func (c *Craw) SmAllWord(qey *goquery.Document) []string {
	list := make([]string, 0)
	list = append(list, c.SmAboutWord(qey)...)
	list = append(list, c.SmOtherWord(qey)...)
	return list
}

//SmSort 根据goquery获取神马首页排序列表
func (c *Craw) SmSort(qey *goquery.Document) []*Site {
	list := make([]*Site, 0)
	qey.Find("#results>.sc").Each(func(id int, g *goquery.Selection) {
		title := g.Find(".c-header-title>span").Eq(0).Text()
		des := g.Find(".js-c-paragraph-text").Eq(0).Text()
		showlink := g.Find(".c-e-source-l>span").First().Text()
		uptime := g.Find(".c-e-source-l>span").Last().Text()
		reallink := g.Find(".c-header-inner[href]").Eq(0).AttrOr("href", "")
		isindex := c.IsIndex(reallink)
		list = append(list, &Site{
			SortID:      id,
			Title:       title,
			Uptime:      uptime,
			Description: des,
			Reallink:    reallink,
			Isindex:     isindex,
			Showlink:    showlink,
			Src:         "神马移动",
		})
	})
	return list
}

//GetRand 获取一个0到max的随机数
func (c *Craw) GetRand(max int) int {
	l := int64(max)
	result, err := rand.Int(rand.Reader, big.NewInt(l))
	if err != nil {
		log.Fatal(err)
	}
	return int(result.Uint64())
}

//RandMemberFromSlice 获取切片的随机值
func (c *Craw) RandMemberFromSlice(s []string) string {
	return s[c.GetRand(len(s)-1)]
}

//GetQey 获取用于goquery对象
func (c *Craw) GetQey(link string, iplist ...string) (*goquery.Document, error) {
	client := c.BaiduClient()
	if len(iplist) != 0 {
		ip := c.RandMemberFromSlice(iplist)
		client = client.Proxy(ip)
	}
	res, body, errs := client.Get(link).End()
	if errs != nil || res.StatusCode != 200 {
		if errs != nil {
			return nil, errs[0]
		}
		return nil, errors.New("返回状态码为" + strconv.Itoa(res.StatusCode))
	}
	defer res.Body.Close()
	bodyreader := strings.NewReader(body)
	qey, err := goquery.NewDocumentFromReader(bodyreader)

	//判断是否是gzip文件
	bodygzip, gziperr := gzip.NewReader(bodyreader)
	if gziperr == nil {
		qey, err = goquery.NewDocumentFromReader(bodygzip)
	}

	if err != nil {
		return nil, err
	}
	return qey, nil
}

//BaiduMobileWordList 根据link返回百度移动端拓展词列表
func (c *Craw) BaiduMobileWordList(link string, iplist ...string) ([]string, error) {
	qey, err := c.GetQey(link, iplist...)
	if err != nil {
		return nil, err
	}
	return c.BaiduMobildAllWord(qey), nil
}

//SmWordList 根据link获取神马拓展词列表
func (c *Craw) SmWordList(link string, iplist ...string) ([]string, error) {
	qey, err := c.GetQey(link, iplist...)
	if err != nil {
		return nil, err
	}
	return c.SmAllWord(qey), nil
}

//BaiduMobileSortList 根据link获得 百度移动排序列表
func (c *Craw) BaiduMobileSortList(link string, iplist ...string) ([]*Site, error) {
	qey, err := c.GetQey(link, iplist...)
	if err != nil {
		return nil, err
	}
	return c.BaiduMobileSort(qey, iplist...), nil
}

//SmSortList 根据link获取神马排序列表
func (c *Craw) SmSortList(link string, iplist ...string) ([]*Site, error) {
	qey, err := c.GetQey(link, iplist...)
	if err != nil {
		return nil, err
	}
	return c.SmSort(qey), nil
}

//BaiduWordAndSort 根据link获取百度拓展词和排序列表
func (c *Craw) BaiduWordAndSort(link string, iplist ...string) ([]string, []*Site, error) {
	qey, err := c.GetQey(link, iplist...)
	if err != nil {
		return nil, nil, err
	}
	return c.BaiduMobildAllWord(qey), c.BaiduMobileSort(qey, iplist...), nil
}

//SmWordAndSort 根据link获取神马拓展词和排序列表
func (c *Craw) SmWordAndSort(link string, iplist ...string) ([]string, []*Site, error) {
	qey, err := c.GetQey(link, iplist...)
	if err != nil {
		return nil, nil, err
	}
	return c.SmAllWord(qey), c.SmSort(qey), nil
}
