package model

import "github.com/dgrijalva/jwt-go"

type (
	ReqBody struct {
		Username string `json:"username" validate:"required"`
		Password string `json:"password"`
	}

	ActivateUserReqBody struct {
		Confirm       string `json:"confirm_password"`
		New           string `json:"new_password"`
		Token         string `json:"token"`
		ActiveUserOtp string `json:"activate_user_otp"`
	}

	AuthForgotPasswordReqBody struct {
		Email string `json:"email" validate:"required"`
	}

	AuthForgotPasswordClaims struct {
		UserID string
		*jwt.StandardClaims
	}

	UserResetPasswordReqBody struct {
		Confirm string `json:"confirm_password"`
		New     string `json:"new_password"`
		Token   string `json:"token"`
		Otp     string `json:"otp"`
	}

	ReqTokenFcm struct {
		TokenFcm string `json:"token_fcm"`
	}
)
