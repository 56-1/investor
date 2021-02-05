package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	URLtools "net/url"
	"regexp"
	"sync"
)

type content struct {
	ErrCode int
	ErrMsg interface{},
	TotalCount int
	PageIndex int
	Data struct{
		LSJZList []Fund
	}
}

type Fund struct {
	FSRQ string
	DWJZ float64
	LJJZ float64
	JZZZL float64
}

var re = regexp.MustCompile(`{.*}`)

func AutoScrapy(task map[string]struct{}, db *sql.DB, RWMu sync.RWMutex){
	limit := make(chan struct{}, 10)

	RWMu.RLock()
	for invest, _ := range task {
		RWMu.RUnlock()
		limit <- struct{]{}
		go Auto(invest, task, limit, db, RWMu)
	}
	RWMu.RUnlock()
}

func Auto(invest string, task map[string]struct{}, ch chan struct{}, db *sql.DB, RWMu sync.RWMutex){
	taskLimit := make(chan struct{}, 50)
	nowPage, totalPage := Scrapy(1, nil, db)

	for ;nowPage <= totalPage; nowPage++ {
		tackLimit <- struct{}{}
		go Scrapy(nowPage, taskLimit, db)
	}

	<-ch
	RWMu.Lock()
	delete(task, invest)
	RWMu.Unlock()
}

func Scrapy(invest string, page int, ch chan struct{}, db *sql.DB)(int, int){
	result, index, total := get(invest, strconv.Itoa(page))
	if result == nil {
		if ch != nil {
			<-ch
		}
		return index, total
	}

	sql := fmt.Sprintf("INSERT INTO %s VALUES(?,?,?,?)", invest)
	s, err := db.Prepare(sql)
	for _, v := range result {
		_, err := s.Exec(v.FSRQ, v.DWJZ, v.LJJZ, v.JZZZL)
		if err != nil {
			continue
		}
	}
	s.Close()

	if ch != nil {
		<-ch
	}

	return index + 1, total / 20
}

func get(invest, page string)([]Fund, int, int){
	url := "http://api.fund.eastmoney.com/f10/lsjz"
	headers = http.Header{}

	timeStamp := strconv.FormatInt(time.Now.UnixNano(), 64)

	params := URLtools.Values{}
	params.Set("callback", "_" + timeStamp)
	params.Set("fundCode", strigs.Replace(invest, "fund_", ""))
	params.Set("pageIndex", page)
	params.Set("pageSize", "20")
	params.Set("startDate", "")
	params.Set("endDate", "")
	params.Set("_", timeStamp)

	url = url + "?" + params.Encode()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err.Error())
		return nil, 0, 0
	}

	req.Header = headers

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err.Error())
		return nil, 0, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println(resp.Status)
		return nil, 0, 0
	}

	t, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err.Error())
		return nil, 0, 0
	}

	data := re.Find(t)
	if data == nil {
		log.Println("it happend error when re was finding data")
		return nil, 0, 0
	}

	c := content{}
	if err = json.UnMarshal(data, &c); err != nil {
		log.Println(err.Error())
		return nil, 0, 0
	}

	if c.ErrCode != 0 {
		log.Println(c.ErrMsg.(string))
		return nil, 0, 0
	}

	return c.Data.LSJZList, c.PageIndex, c.TotalCount
}
