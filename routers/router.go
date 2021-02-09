package routers

import (
	"investment/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	beego.Router("/investment/?:kind:string", &controllers.InvestController{})
	beego.Router("/investment/updateorscrapy", &controllers.ScrapyController{})
	beego.Router("/investment/list/:kind:string", &controllers.ListController{})
}
