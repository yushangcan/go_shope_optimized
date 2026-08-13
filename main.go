package main

import (
	"go_shope/model"

	"github.com/gin-gonic/gin"
)

func main() {
	//导入数据库等配置文件

	//启动gin路由
	service := gin.Default()
	u := &model.User{}
	u.ServicerRouter(service)

}
