package auth

import (
	"erp-cbqa-global/lib/email/template"
	"erp-cbqa-global/lib/env"
	"erp-cbqa-global/lib/smtp"
	"errors"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"erp-cbqa-global/domain/dashboard/auth/model"
	"erp-cbqa-global/domain/dashboard/auth/repository"
	"erp-cbqa-global/lib/constant"
	"erp-cbqa-global/lib/encrypt"

	formLib "erp-cbqa-global/lib/form"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

type AuthServiceInterface interface {
	Login(reqBody model.ReqBody) (*model.ResBody, int, error)
	CheckAuth(string) (*model.AuthUser, error)
	Logout(user model.AuthUser) (int, error)
	ActivateUser(reqBody *model.ActivateUserReqBody) (int, error)
	ForgotPassword(reqBody *model.AuthForgotPasswordReqBody) (model.AuthForgotPasswordResBody, int, error)
	ResetPassword(reqBody *model.UserResetPasswordReqBody) (int, error)
	TokenFcm(id string, req model.ReqTokenFcm) (bool, int, error)
}

type authService struct {
	Repository repository.AuthRepositoryInterface
}

func AuthService(repository repository.AuthRepositoryInterface) AuthServiceInterface {
	return &authService{
		Repository: repository,
	}
}

func (s *authService) Login(reqBody model.ReqBody) (*model.ResBody, int, error) {
	var resBody model.ResBody
	emptyPassword, message := s.checkEmptyEmailPassword("validatePassword", reqBody.Password)
	if emptyPassword {
		return nil, http.StatusBadRequest, errors.New(message)
	}
	user, err := s.Repository.FirstByUsername(reqBody.Username)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New(constant.UserNotFound)
	}
	if user.IsActive == 2 {
		return nil, http.StatusBadRequest, errors.New("user account is unverified")
	}
	if err = encrypt.CompareHashAndPassword(&user.Password, &reqBody.Password); err != nil {
		return nil, http.StatusBadRequest, errors.New(constant.PasswordIsIncorrect)
	}
	authUser := model.AuthUser{
		Username: user.Username,
		ID:       user.ID,
		RoleID:   user.RoleID,
	}
	claims := model.Jwt{
		AuthUser: &authUser,
		StandardClaims: &jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * time.Duration(12)).Unix(),
		},
	}
	jwtToken, err := encrypt.NewWithClaims(&claims)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if err = s.Repository.UpdateToken(user, jwtToken); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	resBody.Token = jwtToken
	resBody.Type = "Bearer"
	resBody.ID = user.ID.String()
	resBody.Username = user.Username
	resBody.Email = user.Email
	resBody.RoleID = user.RoleID.String()
	resBody.Role = user.Role.Name
	return &resBody, http.StatusOK, nil
}

func (s *authService) CheckAuth(bearerToken string) (*model.AuthUser, error) {
	tokenRaw, claims, err := encrypt.Parse(bearerToken)
	if err != nil || claims["RoleID"] == nil { // role id only on user, consumer cannot login
		return nil, errors.New(constant.NotAuthorize)
	}
	id := string(claims["ID"].(string))
	user, err := s.Repository.FirstByID(id)
	if err != nil {
		if err.Error() == constant.RecordNotFound {
			return nil, errors.New(constant.UserNotFound)
		}
	}

	if user.Role.Name == "ROLE_SUPERADMIN" {
		authUser := model.AuthUser{
			ID:       user.ID,
			Username: user.Username,
			RoleID:   user.RoleID,
		}
		if user.Token != tokenRaw {
			return nil, errors.New(constant.UserHasSignedOut)
		}
		return &authUser, nil
	} else {
		authUser := model.AuthUser{
			ID:       user.ID,
			Username: user.Username,
			RoleID:   user.RoleID,
		}
		if user.Token != tokenRaw {
			return nil, errors.New(constant.UserHasSignedOut)
		}
		return &authUser, nil
	}
}

func (s *authService) Logout(authUser model.AuthUser) (int, error) {
	user := model.User{
		ID: authUser.ID,
	}
	if err := s.Repository.DeleteToken(user); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (s *authService) ActivateUser(reqBody *model.ActivateUserReqBody) (int, error) {
	if reqBody.Token == "" {
		return http.StatusBadRequest, errors.New(constant.MissingToken)
	}
	validateOtp, message := s.checkOtpActivateUser(reqBody.ActiveUserOtp, reqBody.Token)
	if !validateOtp {
		return http.StatusBadRequest, errors.New(message)
	}
	if validateOtp && reqBody.New == "" && reqBody.Token != "" {
		return http.StatusOK, errors.New(message)
	}
	emptyPasswordNew, message := s.checkEmptyEmailPassword("validatePassword", reqBody.New)
	if emptyPasswordNew {
		return http.StatusBadRequest, errors.New(message)
	}
	emptyPasswordConfirm, message := s.checkEmptyEmailPassword("validatePassword", reqBody.Confirm)
	if emptyPasswordConfirm {
		return http.StatusBadRequest, errors.New(message)
	}
	if len(reqBody.New) < 8 {
		return http.StatusBadRequest, errors.New(constant.ValidateMinimumPassword)
	}
	if len(reqBody.New) > 16 {
		return http.StatusBadRequest, errors.New(constant.ValidateMaximumPassword)
	}
	checkNewPassword := formLib.ValidatePasswordCMS(reqBody.New)
	if !checkNewPassword {
		return http.StatusBadRequest, errors.New(constant.PasswordRegex)
	}
	tokenRaw, claims, err := encrypt.Parse(reqBody.Token)
	if err != nil {
		return http.StatusBadRequest, err
	}
	userID, _ := uuid.Parse(claims["ID"].(string))
	user, err := s.Repository.QuerySelectUser([]string{"id"}, &model.User{
		ID:                 userID,
		ResetPasswordOtp:   &reqBody.ActiveUserOtp,
		ResetPasswordToken: &tokenRaw,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusBadRequest, errors.New(constant.UserNotFound)
		}
		return http.StatusInternalServerError, err
	}
	if reqBody.New != reqBody.Confirm {
		return http.StatusBadRequest, errors.New(constant.NewAndConfirmPasswordMustBeMatch)
	}
	if err = encrypt.GenerateFromPassword(&reqBody.New); err != nil {
		return http.StatusInternalServerError, err
	}
	userIDStr := user.ID.String()
	if err = s.Repository.UpdateEncryptedPasswordAndActivateUser(&userIDStr, &reqBody.New); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (s *authService) ForgotPassword(reqBody *model.AuthForgotPasswordReqBody) (model.AuthForgotPasswordResBody, int, error) {
	resBody := model.AuthForgotPasswordResBody{}
	emptyEmail, message := s.checkEmptyEmailPassword("validateEmail", reqBody.Email)
	if emptyEmail {
		return resBody, http.StatusBadRequest, errors.New(message)
	}
	checkEmail := s.validateEmailFormat(reqBody.Email)
	if len(checkEmail) < 2 {
		return resBody, http.StatusBadRequest, errors.New(constant.EmailInvalid)
	}
	reqBody.Email = checkEmail[0]
	user, err := s.Repository.QuerySelectUser([]string{"email", "id", "full_name", "otp"}, &model.User{Email: reqBody.Email})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resBody, http.StatusBadRequest, errors.New(constant.UserNotFound)
		}
		return resBody, http.StatusInternalServerError, err
	}
	claims := model.AuthForgotPasswordClaims{
		UserID: user.ID.String(),
		StandardClaims: &jwt.StandardClaims{
			ExpiresAt: time.Now().AddDate(0, 0, 1).Unix(),
		},
	}
	token, err := encrypt.NewWithClaims(claims)
	if err != nil {
		return resBody, http.StatusInternalServerError, err
	}
	rand.Seed(time.Now().UnixNano())
	otp := strconv.Itoa(rand.Int())[:6]
	userIDStr := user.ID.String()
	if err = s.Repository.UpdateForgotPasswordToken(&userIDStr, &token, &otp); err != nil {
		return resBody, http.StatusInternalServerError, err
	}
	body := template.UserForgotPassword
	body = strings.ReplaceAll(body, "{{replace.name}}", user.FullName)
	body = strings.ReplaceAll(body, "{{replace.otp}}", otp)
	body = strings.ReplaceAll(body, "{{replace.url}}", env.String("BASE_URL", "")+"/email-verification?reset_token="+token)
	msg := smtp.SendEmailSmtpRequest{
		To:      []string{user.Username},
		Subject: "Forgot Password - CBQA Global",
		Body:    body,
	}
	resCode, err := smtp.SendEmailSmtp(msg)
	if err != nil {
		return resBody, resCode, err
	}
	resBody.Message = constant.EmailForgotPassword
	return resBody, http.StatusOK, nil
}

func (s *authService) ResetPassword(reqBody *model.UserResetPasswordReqBody) (int, error) {
	if reqBody.Token == "" {
		return http.StatusBadRequest, errors.New(constant.MissingToken)
	}
	validateOtp, message := s.checkOtp(reqBody.Otp, reqBody.Token)
	if !validateOtp {
		return http.StatusBadRequest, errors.New(message)
	}
	if validateOtp && reqBody.New == "" {
		return http.StatusOK, errors.New(message)
	}
	emptyPasswordNew, message := s.checkEmptyEmailPassword("validatePassword", reqBody.New)
	if emptyPasswordNew {
		return http.StatusBadRequest, errors.New(message)
	}
	emptyPasswordConfirm, message := s.checkEmptyEmailPassword("validatePassword", reqBody.Confirm)
	if emptyPasswordConfirm {
		return http.StatusBadRequest, errors.New(message)
	}
	if len(reqBody.New) < 8 {
		return http.StatusBadRequest, errors.New(constant.ValidateMinimumPassword)
	}
	if len(reqBody.New) > 16 {
		return http.StatusBadRequest, errors.New(constant.ValidateMaximumPassword)
	}
	checkNewPassword := formLib.ValidatePasswordCMS(reqBody.New)
	if !checkNewPassword {
		return http.StatusBadRequest, errors.New(constant.PasswordRegex)
	}
	tokenRaw, claims, err := encrypt.Parse(reqBody.Token)
	if err != nil {
		return http.StatusBadRequest, err
	}
	userID, _ := uuid.Parse(claims["ID"].(string))
	user, err := s.Repository.QuerySelectUser([]string{"id"}, &model.User{
		ID:                  userID,
		Otp:                 &reqBody.Otp,
		ForgotPasswordToken: &tokenRaw,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusBadRequest, errors.New(constant.UserNotFound)
		}
		return http.StatusInternalServerError, err
	}
	if reqBody.New != reqBody.Confirm {
		return http.StatusBadRequest, errors.New(constant.NewAndConfirmPasswordMustBeMatch)
	}
	if err = encrypt.GenerateFromPassword(&reqBody.New); err != nil {
		return http.StatusInternalServerError, err
	}
	userIDStr := user.ID.String()
	if err = s.Repository.UpdateEncryptedPasswordAndResetToken(&userIDStr, &reqBody.New); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func (s *authService) validateEmailFormat(email string) []string {
	re := regexp.MustCompile(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`)
	valid_email := re.FindStringSubmatch(email)
	return valid_email
}

func (s *authService) checkEmptyEmailPassword(requestVar string, requestBody string) (bool, string) {
	if requestVar == "validateEmail" {
		if requestBody == "" {
			return true, constant.EmailEmpty
		} else {
			return false, ""
		}
	} else {
		if requestBody == "" {
			return true, constant.PasswordEmpty
		} else {
			return false, ""
		}
	}

}

func (s *authService) checkOtpActivateUser(otp, token string) (bool, string) {
	token_ptr := &token
	user, err := s.Repository.QuerySelectUser([]string{"id", "reset_password_otp"}, &model.User{ResetPasswordToken: token_ptr})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, constant.InvalidToken
		}
	}
	if user.ResetPasswordOtp != nil && *user.ResetPasswordOtp != otp {
		return false, constant.InvalidOtp
	}
	return true, constant.ValidOtp
}

func (s *authService) checkOtp(otp, token string) (bool, string) {
	token_ptr := &token
	user, err := s.Repository.QuerySelectUser([]string{"id", "otp"}, &model.User{ForgotPasswordToken: token_ptr})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, constant.InvalidToken
		}
	}
	if user.Otp != nil && *user.Otp != otp {
		return false, constant.InvalidOtp
	}
	return true, constant.ValidOtp
}

func (s *authService) TokenFcm(id string, req model.ReqTokenFcm) (bool, int, error) {
	if err := s.Repository.UpdateTokenFcm(id, req.TokenFcm); err != nil {
		return false, http.StatusInternalServerError, err
	}
	return true, http.StatusOK, nil
}
