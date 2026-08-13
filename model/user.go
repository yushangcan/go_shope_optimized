package model

import "github.com/gin-gonic/gin"

type User struct {
	id    int
	name  string
	power string
}

func (u *User)ServicerRouter(service *gin.Engine){
	service.POST("/login", u.Login)

}


func(u *User)Login(service *gin.Context){

}