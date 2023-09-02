package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/k0kubun/pp"
	"github.com/sirupsen/logrus"
)

type ApiResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Success   bool        `json:"success"`
	Timestamp int64       `json:"timestamp"`
	Result    interface{} `json:"result"`
}

type LawListResult struct {
	Data       []Law `json:"data"`
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	TotalSizes int   `json:"totalSizes"`
}

type DetailResultLink struct {
	Path string `json:"path"`
	Url  string `json:"url"`
	Type string `json:"type"`
}

type DetailResult struct {
	Body []DetailResultLink `json:"body"`
}

type Law struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Office           string `json:"office"`
	Publish          string `json:"publish"`
	Expiry           string `json:"expiry"`
	Type             string `json:"type"`
	Status           string `json:"status"`
	PageLink         string `json:"url"`
	DownloadLinkWord string `json:"download_link_word"`
	DownloadLinkHtml string `json:"download_link_html"`
	DownloadLinkPdf  string `json:"download_link_pdf"`
	Content          string `json:"content"`
}

const (
	TIMEOUT                    = 30 * time.Second
	NUM_OF_WORKERS_FOR_LISTING = 5
	NUM_OF_WORKERS_FOR_LINKS   = 5
	NUM_OF_WORKERS_FOR_CONTENT = 5
	MAX_PAGE                   = 1000000
)

func get_list_by_page(page int) (*LawListResult, error) {
	url := "https://flk.npc.gov.cn/api/"
	client := &http.Client{Timeout: TIMEOUT}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("type", "")
	q.Add("searchType", "title;vague")
	q.Add("sortTr", "f_bbrq_s;desc")
	q.Add("gbrqStart", "")
	q.Add("gbrqEnd", "")
	q.Add("sxrqStart", "")
	q.Add("sxrqEnd", "")
	q.Add("page", fmt.Sprintf("%d", page))
	q.Add("size", fmt.Sprintf("%d", 10))
	q.Add("_", fmt.Sprintf("%d", time.Now().UnixMilli()))
	req.URL.RawQuery = q.Encode()

	// 设置头信息
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/115.0")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://flk.npc.gov.cn/index.html")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("❌ get_data_from_flknpc(): client.Do(): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("❌ get_data_from_flknpc(): resp.StatusCode = %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("❌ get_data_from_flknpc(): io.ReadAll(): %w", err)
	}
	// var result map[string]interface{}
	var result = ApiResponse{Result: &LawListResult{}}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("❌ get_data_from_flknpc(): json.Unmarshal(): %w", err)
	}
	if result.Code != 200 || !result.Success {
		return nil, fmt.Errorf("❌ get_data_from_flknpc(): %d. [%s]", result.Code, result.Message)
	}

	ret := result.Result.(*LawListResult)

	//	修复一些字段
	for i, law := range ret.Data {
		switch law.Status {
		case "1":
			ret.Data[i].Status = "有效"
		case "3":
			ret.Data[i].Status = "尚未生效"
		case "5":
			ret.Data[i].Status = "已修改"
		case "7":
			// 修改、废止的决定
		case "9":
			ret.Data[i].Status = "已废止"
		}
	}

	return ret, nil

}

func get_links(law Law) (map[string]string, error) {
	id := law.ID
	link := "https://flk.npc.gov.cn/api/detail"
	client := &http.Client{Timeout: TIMEOUT}
	data := url.Values{}
	data.Set("id", fmt.Sprint(id))

	resp, err := client.PostForm(link, data)
	if err != nil {
		return nil, fmt.Errorf("❌ get_links(): client.PostForm(): 《%s》 %w", law.Title, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("❌ get_links(): resp.StatusCode = 《%s》 %d", law.Title, resp.StatusCode)
	}

	var result = ApiResponse{Result: &DetailResult{}}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("❌ get_links(): json.NewDecoder().Decode(): 《%s》 %w", law.Title, err)
	}

	if result.Code != 200 || !result.Success {
		return nil, fmt.Errorf("❌ %d. [%s]", result.Code, result.Message)
	}

	detail := make(map[string]string)
	for _, link := range result.Result.(*DetailResult).Body {
		l := link.Path
		if len(l) == 0 {
			l = link.Url
		}
		detail[link.Type] = fmt.Sprintf("https://wb.flk.npc.gov.cn%s", l)
	}

	return detail, nil
}

func get_content(law Law) (string, error) {
	//  获取内容链接、格式
	var link, from, to string
	if len(law.DownloadLinkWord) > 0 {
		link = law.DownloadLinkWord
		from = "docx"
		to = "markdown"
	} else if len(law.DownloadLinkHtml) > 0 {
		link = law.DownloadLinkHtml
		from = "html"
		to = "plain"
	} else if len(law.DownloadLinkPdf) > 0 {
		link = law.DownloadLinkPdf
		from = "pdf"
		to = "markdown"
	} else {
		return "", fmt.Errorf("❌ get_content(): 《%s》 no content link", law.Title)
	}

	//	下载文件
	resp, err := http.Get(link)
	if err != nil {
		return "", fmt.Errorf("❌ get_content(): http.Get(): 《%s》 %w", law.Title, err)
	}
	defer resp.Body.Close()
	tempFile, err := os.CreateTemp("", "flk.crawler.*")
	if err != nil {
		return "", fmt.Errorf("❌ get_content(): os.CreateTemp(): 《%s》 %w", law.Title, err)
	}
	defer os.Remove(tempFile.Name())
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("❌ get_content(): io.Copy(): 《%s》 %w", law.Title, err)
	}
	tempFile.Close()

	//	转换格式
	cmd := exec.Command("pandoc", "-f", from, "-t", to, tempFile.Name()) //, "--extract-media", ".")
	output, err := cmd.Output()
	if err == nil {
		return string(output), nil
	}

	//	如果转换失败，换 tika 来转换
	logrus.Warnf("⚠️ 《%s》 pandoc 转换失败，正尝试 tika 转换", law.Title)
	cmd = exec.Command("tika", "-t", tempFile.Name())
	output, err = cmd.Output()
	if err != nil {
		//	如果 tika 也失败了，就返回错误
		pp.Println(law)
		fmt.Printf("> cmd = %s\n", cmd.String())
		content, _ := os.ReadFile(tempFile.Name())
		fmt.Printf("> content\n----------------\n%s\n----------------\n", content[:1000])
		return "", fmt.Errorf("❌ get_content(): cmd.Output(): 《%s》 %w", law.Title, err)
	}
	return string(output), nil
}

func load(filename string) ([]Law, error) {
	laws := []Law{}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&laws)
	if err != nil {
		return nil, err
	}
	return laws, nil
}

func save(filename string, laws []Law) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(laws)
	if err != nil {
		return err
	}
	return nil
}

func startWorkders(name string, workerCount int, handler func(), closer func()) {
	logrus.Debugf("正在为 「%s」 任务启动 %d 个协程", name, workerCount)
	wg := sync.WaitGroup{}
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler()
		}()
	}
	if closer != nil {
		go func() {
			wg.Wait()
			logrus.Debugf("正在关闭「%s」任务", name)
			closer()
		}()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	chPage := make(chan int, 10)
	chLaw := make(chan Law, 10)
	chLink := make(chan Law, 10)
	chContent := make(chan Law, 10)
	chSave := make(chan []Law, 10)
	oldTotalSizes := 0
	totalSizes := 200
	totalPages := 20
	start := time.Now()
	dbfile := "laws.json"

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	//	准备日志文件
	os.MkdirAll("logs", 0755)
	logfilename := fmt.Sprintf("logs/crawler.%s.log", start.Format("2006-01-02_1504"))
	logfile, err := os.Create(logfilename)
	if err != nil {
		logrus.Errorf("❌ os.Create(): %s", err)
		return
	}
	defer logfile.Close()
	mw := io.MultiWriter(os.Stdout, logfile)
	logrus.SetOutput(mw)

	logrus.Info("开始爬取「国家法律法规数据库」网站")

	//	加载历史记录
	total_laws, err := load(dbfile)
	if err != nil {
		logrus.Errorf("❌ load(): %s", err)
		total_laws = []Law{}
	}
	oldTotalSizes = len(total_laws)
	logrus.Infof("已经获取 %d 条法规", oldTotalSizes)

	exists_laws_map := make(map[string]int)
	for i, law := range total_laws {
		exists_laws_map[law.ID] = i
	}

	//	向 chPage 中添加页码
	go func() {
		for page := 1; page <= min(totalPages, MAX_PAGE); page++ {
			chPage <- page
		}
		close(chPage)
	}()

	//	获取一页信息
	startWorkders("获取列表页", NUM_OF_WORKERS_FOR_LISTING, func() {
		for page := range chPage {
			result, err := get_list_by_page(page)
			if err != nil {
				logrus.Errorf("❌ get_list_by_page(): %s", err)
				continue
			}
			if result.TotalSizes > totalSizes {
				totalSizes = result.TotalSizes
				totalPages = totalSizes/10 + 1
			}
			laws := result.Data
			for _, law := range laws {
				chLaw <- law
			}
		}
	}, func() { close(chLaw) })

	//	获取链接
	startWorkders("获取法规链接", NUM_OF_WORKERS_FOR_LINKS, func() {
		for law := range chLaw {
			if id, exists := exists_laws_map[law.ID]; exists {
				exist_law := total_laws[id]
				if len(exist_law.DownloadLinkWord+exist_law.DownloadLinkHtml+exist_law.DownloadLinkPdf) > 0 {
					//	对于已经有下载链接的，则不必再获取，直接传给下一步
					chLink <- exist_law
					continue
				}
			}
			links, err := get_links(law)
			if err != nil {
				logrus.Errorf("❌ get_links(): %s", err)
				continue
			}
			for k, v := range links {
				switch k {
				case "WORD":
					law.DownloadLinkWord = v
				case "HTML":
					law.DownloadLinkHtml = v
				case "PDF":
					law.DownloadLinkPdf = v
				}
			}
			chLink <- law
		}
	}, func() { close(chLink) })

	//	获取内容
	startWorkders("获取法规内容", NUM_OF_WORKERS_FOR_CONTENT, func() {
		for law := range chLink {
			if id, exists := exists_laws_map[law.ID]; exists {
				exist_law := total_laws[id]
				if len(exist_law.Content) > 0 {
					//	对于已经有内容的，并且也在列表里的，则直接跳过
					chContent <- law
					continue
				}
			}
			content, err := get_content(law)
			if err != nil {
				logrus.Errorf("❌ get_content(): %s", err)
				continue
			}
			law.Content = content
			chContent <- law
		}
	}, func() { close(chContent) })

	//	保存文件
	go func() {
		for law := range chContent {
			progress_percent := float32((len(total_laws) * 100.00) / totalSizes)
			if id, exists := exists_laws_map[law.ID]; exists {
				exist_law := total_laws[id]
				if len(exist_law.Content) > 0 {
					//	对于已经有内容的，并且也在列表里的，则直接跳过
					logrus.Infof("(%.2f%%) [%5d] 「%s」: 《%s》 <已存在>", progress_percent, id, law.Type, law.Title)
					continue
				}
			}

			logrus.Infof("(%.2f%%) [%5d] 「%s」: 《%s》", progress_percent, len(total_laws), law.Type, law.Title)
			total_laws = append(total_laws, law)

			if len(total_laws)%10 == 0 {
				err := save(dbfile, total_laws)
				if err != nil {
					logrus.Errorf("❌ save(): %s", err)
					return
				}
			}
		}
		err := save(dbfile, total_laws)
		if err != nil {
			logrus.Errorf("❌ save(): %s", err)
			return
		}
		chSave <- total_laws
		close(chSave)
	}()

	//	等待所有协程结束
	for range chSave {
	}

	logrus.Info("Done!")
	logrus.Infof("总共用时: %s，开始于 %s，结束于 %s", time.Since(start), start.Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02 15:04:05"))
	logrus.Infof("总共获取 %d 条法规（网站总数为 %d），此次获取 %d 条法规", len(total_laws), totalSizes, len(total_laws)-oldTotalSizes)
}
