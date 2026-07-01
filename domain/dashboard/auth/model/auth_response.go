package model

type (
	ResBody struct {
		Token    string `json:"token"`
		Type     string `json:"type"`
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		RoleID   string `json:"role_id"`
		Role     string `json:"role"`
	}

	AuthForgotPasswordResBody struct {
		Message string `json:"message"`
	}
)
