package controllers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/cache"
	"github.com/astaxie/beego/validation"

	_ "github.com/go-sql-driver/mysql"
)

type data struct {
	Key  string `json:"key"`
	Code string `json:"code"`
}

type msg struct {
	Status  bool   `json:"status"`
	Message string `json:"msg"`
}

type content struct {
	Now   float64   `json:"now"`
	Mean  float64   `json:"mean"`
	Std   float64   `json:"std"`
	Poins []float64 `json:"poins"`
}

type List struct {
	Date  string  `json:"date"`
	Price float64 `json:"price"`
	Ratio float64 `json:"ratio"`
}

type MainController struct {
	beego.Controller
}

type InvestController struct {
	beego.Controller
}

type ScrapyController struct {
	beego.Controller
}

type ListController struct {
	beego.Controller
}

var bm cache.Cache
var db *sql.DB
var RWMu *sync.RWMutex = &sync.RWMutex{}
var task = map[string]struct{}{}

func init() {
	var err error
	bm, err = cache.NewCache("memory", `{"interval": 43200}`)
	if err != nil {
		log.Fatalln(err.Error())
	}

	user := beego.AppConfig.String("mysqluser")
	pass := beego.AppConfig.String("mysqlpass")
	ip := beego.AppConfig.String("mysqlip")
	port := beego.AppConfig.String("mysqlport")
	dbName := beego.AppConfig.String("mysqldbname")

	conf := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", user, pass, ip, port, dbName)
	db, err = sql.Open("mysql", conf)
	if err != nil {
		log.Fatalln(err.Error())
	}

	go AutoScrapy(task, db, RWMu)
}

func (m *MainController) Get() {
	m.Redirect("/investment", 302)
}

func (i *InvestController) Get() {
	i.Layout = "layout.html"
	i.LayoutSections = make(map[string]string)

	key, err := PublicKey()
	if err != nil {
		i.LayoutSections["HTMLHead"] = ""
		i.Data["Message"] = err.Error()
		i.TplName = "error.html"
		return
	}

	i.LayoutSections["HTMLHead"] = "htmlhead.html"
	i.Data["PublicKey"] = key
	i.TplName = "index.html"
}

func (i *InvestController) Post() {
	kind := i.Ctx.Input.Param(":kind")

	d := data{}
	if err := json.Unmarshal(i.Ctx.Input.RequestBody, &d); err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	valid := validation.Validation{}
	valid.Base64(d.Key, "key")
	valid.Base64(d.Code, "code")
	valid.MaxSize(d.Code, 8, "code length")
	valid.Required(kind, "kind")

	if valid.HasErrors() {
		//valid.Errors{key:message}
		mes, err := json.Marshal(valid.Errors)
		if err != nil {
			mes = []byte(err.Error())
		}
		m := msg{Status: false, Message: string(mes)}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	cipherKey, err := base64.StdEncoding.DecodeString(d.Key)
	if err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	key, err := RSADecrypt(cipherKey)
	if err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	cipherCode, err := base64.StdEncoding.DecodeString(d.Code)
	if err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	codeByte, err := AESDecryptCFB(key, cipherCode)
	if err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	code := string(codeByte)
	valid.Length(code, 6, "code")
	if valid.HasErrors() {
		mes, err := json.Marshal(valid.Errors)
		if err != nil {
			mes = []byte(err.Error())
		}
		m := msg{Status: false, Message: string(mes)}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	//use kind and code
	invest := fmt.Sprintf("%s_%s", kind, code)

	RWMu.RLock()
	if _, ok := task[invest]; ok {
		RWMu.RUnlock()
		m := msg{Status: false, Message: fmt.Sprintf("add %s to task, Please waitting for scrapy data", invest)}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}
	RWMu.RUnlock()

	if bm.IsExist(invest) {
		content := bm.Get(invest).(string) //storage string
		m := msg{Status: true, Message: content}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	//handle update or scrapy?
	var now float64
	var date string
	today := time.Now()
	if today.Weekday() == time.Sunday {
		date = today.AddDate(0, 0, -2).Format("2006-01-02")
	} else if today.Weekday() == time.Monday {
		date = today.AddDate(0, 0, -3).Format("2006-01-02")
	} else {
		date = today.AddDate(0, 0, -1).Format("2006-01-02")
	}
	rsql := fmt.Sprintf("SELECT price FROM %s WHERE date=?", invest)
	err = db.QueryRow(rsql, date).Scan(&now)
	if err != nil {
		if strings.Contains(err.Error(), "exist") {
			flash := beego.NewFlash()
			flash.Set("scrapy", invest)
			flash.Store(&i.Controller)
			i.Redirect("/investment/updateorscrapy", 302)
			return
		}

		if err == sql.ErrNoRows {
			//handle update
			flash := beego.NewFlash()
			flash.Set("update", invest)
			flash.Store(&i.Controller)
			i.Redirect("/investment/updateorscrapy", 302)
			return
		}
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	rsql = fmt.Sprintf("SELECT price FROM %s", invest)
	rows, err := db.Query(rsql)
	if err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}
	defer rows.Close()

	x := make([]float64, 0)
	for rows.Next() {
		var k float64
		err = rows.Scan(&k)
		if err != nil {
			m := msg{Status: false, Message: err.Error()}
			t, _ := json.Marshal(&m)
			i.Ctx.WriteString(string(t))
			return
		}

		x = append(x, k)
	}

	mean, std := MeanAndStd(x)
	poins := Collect(x)

	c := content{Now: now, Mean: mean, Std: std, Poins: poins}
	t, err := json.Marshal(&c)
	if err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ = json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	bm.Put(invest, string(t), 21600)
	m := msg{Status: true, Message: string(t)}
	t, _ = json.Marshal(&m)
	i.Ctx.WriteString(string(t))
	return
}

func (s *ScrapyController) Get() {
	flash := beego.ReadFromRequest(&s.Controller)
	if invest, ok := flash.Data["scrapy"]; ok {
		//handle scrapy
		RWMu.Lock()
		task[invest] = struct{}{}
		RWMu.Unlock()

		m := msg{Status: false, Message: fmt.Sprintf("add %s to task, Please waitting for scrapy data", invest)}
		t, _ := json.Marshal(&m)
		s.Ctx.WriteString(string(t))
		return
	} else if invest, ok = flash.Data["update"]; ok {
		//handle update
		Auto(invest, nil, nil, db, RWMu)
		m := msg{Status: false, Message: "update complete, Please refresh"}
		t, _ := json.Marshal(&m)
		s.Ctx.WriteString(string(t))
		return
	} else {
		m := msg{Status: false, Message: "flash error"}
		t, _ := json.Marshal(&m)
		s.Ctx.WriteString(string(t))
	}

	return
}

func (l *ListController) Get() {
	kind := l.Ctx.Input.Param(":kind")

	rsql := fmt.Sprintf("SELECT date, price, ratio FROM %s ORDER BY date DESC LIMIT 0, 7", kind)

	rows, err := db.Query(rsql)
	if err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		l.Ctx.WriteString(string(t))
		return
	}

	list := []List{}
	tl := List{}

	for rows.Next() {
		err := rows.Scan(&tl.Date, &tl.Price, &tl.Ratio)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		list = append(list, tl)
	}

	t, _ := json.Marshal(&list)
	m := msg{Status: true, Message: string(t)}
	t, _ = json.Marshal(&m)
	l.Ctx.WriteString(string(t))
	return
}
