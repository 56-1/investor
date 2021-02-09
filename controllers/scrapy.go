package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	URLtools "net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Resp struct {
	ErrCode    int
	ErrMsg     interface{}
	TotalCount int
	PageIndex  int
	Data       struct {
		LSJZList []Fund
	}
}

type Fund struct {
	FSRQ  string
	DWJZ  string
	LJJZ  string
	JZZZL string
}

var re = regexp.MustCompile(`{.*}`)

func AutoScrapy(task map[string]struct{}, db *sql.DB, RWMu *sync.RWMutex) {
	rsql := "CREATE TABLE IF NOT EXISTS %s (date DATE PRIMARY KEY, price DOUBLE, total_price DOUBLE, ratio DOUBLE)"
	limit := make(chan struct{}, 10)

	for {
		RWMu.RLock()
		for invest := range task {
			RWMu.RUnlock()
			rsql = fmt.Sprintf(rsql, invest)
			_, err := db.Exec(rsql)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			limit <- struct{}{}
			go Auto(invest, task, limit, db, RWMu)

			RWMu.RLock()
		}
		RWMu.RUnlock()
		time.Sleep(10 * time.Second)
	}
}

func Auto(invest string, task map[string]struct{}, ch chan struct{}, db *sql.DB, RWMu *sync.RWMutex) {
	taskLimit := make(chan struct{}, 50)
	nowPage, totalPage := Scrapy(invest, 1, nil, db)

	if task == nil {
		return
	}

	for ; nowPage <= totalPage; nowPage++ {
		taskLimit <- struct{}{}
		go Scrapy(invest, nowPage, taskLimit, db)
	}

	<-ch
	RWMu.Lock()
	delete(task, invest)
	RWMu.Unlock()
}

func Scrapy(invest string, page int, ch chan struct{}, db *sql.DB) (int, int) {
	result, index, total := get(invest, strconv.Itoa(page))
	if result == nil {
		if ch != nil {
			<-ch
		}
		return index, total
	}

	sql := fmt.Sprintf("INSERT INTO %s VALUES(?,?,?,?)", invest)
	s, err := db.Prepare(sql)
	if err != nil {
		log.Println(err.Error())
		return 0, 0
	}
	for _, v := range result {
		dwjz, err2 := strconv.ParseFloat(v.DWJZ, 64)
		ljjz, err3 := strconv.ParseFloat(v.LJJZ, 64)
		jzzzl, err4 := strconv.ParseFloat(v.JZZZL, 64)
		if err2 != nil || err3 != nil || err4 != nil {
			log.Println("Format string to float64 happend error")
			continue
		}
		_, err := s.Exec(v.FSRQ, dwjz, ljjz, jzzzl)
		if err != nil {
			if !strings.Contains(err.Error(), "PRIMARY") {
				log.Println(err.Error())
			}
			continue
		}
	}
	s.Close()

	if ch != nil {
		<-ch
	}

	return index + 1, total / 20
}

func get(invest, page string) ([]Fund, int, int) {
	url := "http://api.fund.eastmoney.com/f10/lsjz"
	headers := http.Header{
		"User-Agent": []string{`Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.80 Safari/537.36`},
		"Referer":    []string{`http://fundf10.eastmoney.com/jjjz_161725.html`},
	}

	timeStamp := strconv.FormatInt(time.Now().UnixNano(), 10)

	params := URLtools.Values{}
	params.Set("callback", "_"+timeStamp)
	params.Set("fundCode", strings.Replace(invest, "fund_", "", -1))
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

	c := Resp{}
	if err = json.Unmarshal(data, &c); err != nil {
		log.Println(err.Error())
		return nil, 0, 0
	}

	if c.ErrCode != 0 {
		log.Println(c.ErrMsg.(string))
		return nil, 0, 0
	}

	return c.Data.LSJZList, c.PageIndex, c.TotalCount
}
