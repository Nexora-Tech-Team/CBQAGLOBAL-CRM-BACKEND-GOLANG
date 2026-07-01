package model

type ReqBody struct {
	Username string `binding:"required,email"`
	Password string `binding:"required,gte=6"`
}
