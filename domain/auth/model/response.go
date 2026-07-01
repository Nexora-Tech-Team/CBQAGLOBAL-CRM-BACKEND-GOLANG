package model

type ResBody struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	RoleID   string `json:"role_id"`
	Token    string `json:"token"`
}
