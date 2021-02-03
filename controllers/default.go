package controllers

import (
	"database/sql"
	"encoding/json"
	"encoding/base64"
	"log"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/cache"
	"github.com/astaxie/beego/validation"

	_ "github.com/go-sql-driver/mysql"
)

type data struct {
	Key string `json:"key"`
	Code string `json:"code"`
}

type msg struct {
	Status bool `json:"status"`
	Message string `json:"msg"`
}

type content struct {
	Std float64 `json:"std"`
	Poins []float64 `json:"poins"`
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

var bm cache.Cache
var db *sql.DB

func init(){
	var err error
	bm, err = chche.NewCache("memory", `{"interval": 43200}`)
	if err != nil {
		log.Fatalln(err.Error())
	}

	user := beego.AppConfig.String("mysqluser")
	pass := beego.AppConfig.String("mysqlpass")
	ip := beego.AppConfig.String("mysqlip")
	port := beego.AppConfig.String("mysqlport")
	dbName := beego.AppConfig.String("dbname")

	conf := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", user, pass, ip, port, dbName)
	db, err = sql.Open("mysql", conf)
	if err != nil {
		log.Fatalln(err.Error())
	}
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
	if valid.HasErrors(){
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

	if bm.IsExist(invest) {
		content := bm.Get(invest).(string)	//storage string
		i.Ctx.WriteString(content)
		return
	}

	if _, err = db.Query("SELECT table_name FROM information_schema WHERE table_name=?", invest); err != nil {
		if err == sql.ErrNoRows {
			flash := beego.NewFlash()
			flash.Set("scrapy", invest)
			flash.Store(&i.Controller)
			i.Redirect("/investment/scrapy")
			return
		}
		m := msg{Status:false, Message:err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	//handle update?
	date := time.Now().Format("2021-01-01")
	sql := fmt.Sprintf("SELECT date FROM %s WHERE date=%s", invest, date)
	if _, err := db.Query(sql); err != nil {
		if err == sql.ErrNoRows {
			//handle update
			flash := beego.NewFlash()
			flash.Set("update", invest)
			flash.Store(&i.Controller)
			i.Redirect("/investment/scrapy")
			return
		}
		m := msg{Status:false, Message:err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	sql = fmt.Sprintf("SELECT price FROM %s", invest)
	rows, err := db.Query(sql)
	if err != nil {
		m := msg{Status:false, Message:err.Error()}
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
			m := msg{Status:false, Message:err.Error()}
			t, _ := json.Marshal(&m)
			i.Ctx.WriteString(string(t))
			return
		}

		x = append(x, k)
	}

	mean, std := MeanAndStd(x)
	poins := Collect(x)

	c := content{Std: std, Poins: poins}
	t, err := json.Marshal(&c)
	if err != nil {
		m := msg{Status:false, Message:err.Error()}
		t, _ = json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	bm.Put(invest, string(t), 21600)
	m := msg{Status:true, Message:string(t)}
	t, _ = json.Marshal(&m)
	i.Ctx.WriteString(string(t))
	return
}

func (s *ScrapyController) Get(){
	flash := beego.ReadFromRequest(&s.Controller)
	if invest, ok := flash.Data["scrapy"]; ok {
		//handle scrapy
	} else if invest, ok = flash.Data["update"]; ok {
		//handle update
	} else {
		m := msg{Status:false, Message:"flash error"}
		t, _ := json.Marshal(&m)
		s.Ctx.WriteString(t)
		return
	}
}
