package controllers

import (
	"encoding/json"
	"encoding/base64"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/validation"
)

type data struct {
	Key string `json:"key"`
	Code string `json:"code"`
}

type msg struct {
	Status bool `json:"status"`
	Message string `json:"msg"`
}

type MainController struct {
	beego.Controller
}

type InvestController struct {
	beego.Controller
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
	valid.Required(d.Key, "key")
	valid.Required(d.Code, "code")
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

	code, err := AESDecryptCFB(key, cipherCode)
	if err != nil {
		m := msg{Status: false, Message: err.Error()}
		t, _ := json.Marshal(&m)
		i.Ctx.WriteString(string(t))
		return
	}

	m := msg{Status: true, Message: string(key)+kind+string(code)}
	t, _ := json.Marshal(&m)
	i.Ctx.WriteString(string(t))
	return

}
