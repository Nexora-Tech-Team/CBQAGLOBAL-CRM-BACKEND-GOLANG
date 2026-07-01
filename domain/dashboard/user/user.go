package user

import (
	"encoding/json"
	"erp-cbqa-global/domain/dashboard/user/model"
	"erp-cbqa-global/domain/dashboard/user/repository"
	authLib "erp-cbqa-global/lib/auth"
	"erp-cbqa-global/lib/constant"
	"erp-cbqa-global/lib/email/template"
	"erp-cbqa-global/lib/encrypt"
	"erp-cbqa-global/lib/env"
	"erp-cbqa-global/lib/smtp"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

const LogHistoryModelName = "User"

type UserServiceInterface interface {
	GetMyProfile(userId string) (model.ResponseUser, int, error)
	CreateUser(reqBody *model.RequestCreateUser, userAuth *authLib.AuthData) (*model.User, int, error)
	GetUserNoLimit(search, status, roleId, sortBy, sort string, isShowDelete bool, limit int) ([]model.ResponseUser, int, error)
	GetUsers(search, status, roleId, sortBy, sort string, offset, limit int, isShowDelete bool) ([]model.ResponseUser, int64, int, error)
	GetDetailUser(id string) (*model.ResponseUser, int, error)
	UpdateUser(id string, req *model.RequestUpdateUser, userAuth *authLib.AuthData) (*model.ResponseUser, int, error)
	DeleteUser(id string, userAuth *authLib.AuthData) (int, error)
	ResendEmailVerification(email string) (int, error)
	ValidateEmail(email string) (int, error)
}

type userService struct {
	Repository repository.UserRepositoryInterface
}

func UserService(repository repository.UserRepositoryInterface) UserServiceInterface {
	return &userService{
		Repository: repository,
	}
}

func (s *userService) GetMyProfile(userId string) (model.ResponseUser, int, error) {
	var data model.ResponseUser
	data, err := s.Repository.GetMyProfile(userId)
	if err != nil {
		return data, http.StatusInternalServerError, nil

	}
	return data, http.StatusOK, nil
}

func (s *userService) CreateUser(reqBody *model.RequestCreateUser, userAuth *authLib.AuthData) (*model.User, int, error) {
	// Parse and capitalize names
	firstName, lastName := parseAndCapitalizeName(reqBody.FullName)
	capitalizedFullName := capitalizeFullName(reqBody.FullName)

	// Hash password
	hashedPassword := reqBody.Password
	if err := encrypt.GenerateFromPassword(&hashedPassword); err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to hash password")
	}

	paramsCreateUser := model.CreateUser{
		Username:       reqBody.Username,
		Password:       hashedPassword,
		EmpId:          reqBody.EmpId,
		RoleId:         reqBody.RoleId,
		ProfileImage:   reqBody.ProfileImage,
		SignatureImage: reqBody.SignatureImage,
		CompanyId:      reqBody.CompanyId,
		FullName:       capitalizedFullName,
		Email:          reqBody.Email,
		Phone:          reqBody.Phone,
		Description:    reqBody.Description,
		IsActive:       2,
		CreatedBy:      userAuth.Username,
		UpdatedBy:      userAuth.Username,
	}

	resUser, resCode, err := s.Repository.CreateUser(paramsCreateUser, firstName, lastName)
	if err != nil {
		if err.(*pq.Error).Code == "23505" { // NEW method for unique key error message
			return nil, http.StatusConflict, errors.New(constant.EmailConflict)
		}
		if err.(*pq.Error).Code == "22P02" { // NEW method for invalid reference foreign key
			return nil, http.StatusBadRequest, errors.New(constant.RoleIdEmpty)
		}
		return nil, resCode, err
	}

	rand.Seed(time.Now().UnixNano())
	otp := strconv.Itoa(rand.Int())[:6]
	claims := model.UserActivationClaims{
		Id:             resUser.Id,
		Otp:            otp,
		StandardClaims: &jwt.StandardClaims{},
	}
	token, err := encrypt.NewWithClaims(claims)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if err = s.Repository.QueryUpdateValuesById(&resUser.Id, map[string]interface{}{
		"reset_password_token": token,
		"reset_password_otp":   otp,
	}); err != nil {
		return nil, http.StatusBadRequest, err
	}

	body := template.UserVerifyAccount
	body = strings.ReplaceAll(body, "{{replace.otp}}", otp)
	body = strings.ReplaceAll(body, "{{replace.url}}", env.String("BASE_URL", "")+"/email-verification?activation_token="+token)
	msg := smtp.SendEmailSmtpRequest{
		To:      []string{reqBody.Email},
		Subject: "Account Activation - CBQA Global",
		Body:    body,
	}
	if env.Bool("EMAIL_MUST_SUCCESS", false) {
		resCode, err = smtp.SendEmailSmtp(msg)
		if err != nil {
			return nil, resCode, errors.New("failed to send verification email")
		}
	} else {
		sendVerificationEmailAsync(msg)
	}

	newObject, err := json.Marshal(map[string]any{
		"id":        resUser.Id,
		"user_name": paramsCreateUser.Username,
		"full_name": reqBody.FullName,
		"phone":     reqBody.Phone,
		"email":     reqBody.Email,
		"role_id":   reqBody.RoleId,
		"status":    paramsCreateUser.IsActive,
	})
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	paramsUserLogHistory := model.UserLogHistory{
		Action:    "create " + resUser.Username,
		Model:     LogHistoryModelName,
		NewObject: newObject,
		ObjectId:  "",
		UserId:    userAuth.ID,
	}

	err = s.Repository.CreateUserLogHistory(paramsUserLogHistory)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	return resUser, http.StatusOK, nil
}

func (s *userService) GetUserNoLimit(search, status, roleId, sortBy, sort string, isShowDelete bool, limit int) ([]model.ResponseUser, int, error) {
	var users []model.ResponseUser
	users, err := s.Repository.GetUserNoLimit(search, status, roleId, sortBy, sort, isShowDelete, limit)
	if err != nil {
		return users, http.StatusBadRequest, err
	}
	return users, http.StatusOK, nil
}

func (s *userService) GetUsers(search, status, roleId, sortBy, sort string, offset, limit int, isShowDelete bool) ([]model.ResponseUser, int64, int, error) {
	var exportHub []model.ResponseUser
	exportHub, totalRow, err := s.Repository.GetUsers(search, status, roleId, sortBy, sort, offset, limit, isShowDelete)
	if err != nil {
		return exportHub, totalRow, http.StatusInternalServerError, err
	}
	return exportHub, totalRow, http.StatusOK, nil
}

func (s *userService) GetDetailUser(id string) (*model.ResponseUser, int, error) {
	data, err := s.Repository.QuerySelectUser([]string{"musers.*"}, &model.User{Id: id})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusBadRequest, gorm.ErrRecordNotFound
		}
		return nil, http.StatusInternalServerError, err
	}
	user := model.ResponseUser{
		Id:             data.Id,
		Code:           data.Code,
		Description:    data.Description,
		Username:       data.Username,
		EmpId:          data.EmpId,
		RoleId:         data.RoleId,
		ProfileImage:   data.ProfileImage,
		SignatureImage: data.SignatureImage,
		IsActive:       data.IsActive,
		CompanyId:      data.CompanyId,
		FullName:       data.FullName,
		FirstName:      data.FirstName,
		LastName:       data.LastName,
		Email:          data.Email,
		Phone:          data.Phone,
	}
	return &user, http.StatusOK, nil
}

func (s *userService) UpdateUser(id string, req *model.RequestUpdateUser, userAuth *authLib.AuthData) (*model.ResponseUser, int, error) {
	data, err := s.Repository.QuerySelectUser([]string{"*"}, &model.User{Id: id})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusBadRequest, gorm.ErrRecordNotFound
		}
		return nil, http.StatusInternalServerError, err
	}

	firstName, lastName := parseAndCapitalizeName(req.FullName)
	capitalizedFullName := capitalizeFullName(req.FullName)

	paramsUpdateUser := model.UpdateUser{
		Id:             id,
		Description:    req.Description,
		Username:       req.Username,
		EmpId:          req.EmpId,
		RoleId:         req.RoleId,
		ProfileImage:   req.ProfileImage,
		SignatureImage: req.SignatureImage,
		CompanyId:      req.CompanyId,
		FullName:       capitalizedFullName,
		FirstName:      firstName,
		LastName:       lastName,
		Email:          req.Email,
		Phone:          req.Phone,
		UpdatedBy:      userAuth.Username,
	}

	if err := s.Repository.UpdateUser(paramsUpdateUser); err != nil {
		if err.(*pq.Error).Code == "23505" { // NEW method for unique key error message
			return nil, http.StatusBadRequest, errors.New(constant.EmailConflict)
		}
		if err.(*pq.Error).Code == "23503" { // NEW method for invalid reference foreign key
			return nil, http.StatusBadRequest, errors.New(constant.InvalidRegionId)
		}
		return nil, http.StatusBadRequest, err
	}

	newObject, err := json.Marshal(map[string]any{
		"username":    req.Username,
		"full_name":   req.FullName,
		"email":       req.Email,
		"status":      req.IsActive,
		"phone":       req.Phone,
		"role_id":     req.RoleId,
		"emp_id":      req.EmpId,
		"company_id":  req.CompanyId,
		"description": req.Description,
	})
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	oldObject, err := json.Marshal(map[string]any{
		"username":    data.Username,
		"full_name":   data.FullName,
		"email":       data.Email,
		"status":      data.IsActive,
		"phone":       data.Phone,
		"role_id":     data.RoleId,
		"emp_id":      data.EmpId,
		"company_id":  data.CompanyId,
		"description": data.Description,
	})
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	paramsLogHistory := model.UserLogHistory{
		UserId:    userAuth.ID,
		ObjectId:  id,
		Action:    "update",
		Model:     LogHistoryModelName,
		NewObject: newObject,
		OldObject: oldObject,
	}
	if err = s.Repository.CreateUserLogHistory(paramsLogHistory); err != nil {
		return nil, http.StatusBadRequest, err
	}

	resUser, err := s.Repository.QuerySelectUser([]string{"musers.*"}, &model.User{Id: id})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusBadRequest, gorm.ErrRecordNotFound
		}
		return nil, http.StatusInternalServerError, err
	}
	user := model.ResponseUser{
		Id:       resUser.Id,
		RoleId:   resUser.RoleId,
		FullName: resUser.FullName,
		Username: resUser.Username,
	}

	return &user, http.StatusOK, nil
}

func (s *userService) DeleteUser(id string, userAuth *authLib.AuthData) (int, error) {
	var paramsLogHistory model.UserLogHistory
	data, err := s.Repository.QuerySelectUser([]string{"*"}, &model.User{Id: id})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusBadRequest, gorm.ErrRecordNotFound
		}
		return http.StatusInternalServerError, err
	}
	err = s.Repository.DeleteById(id)
	if err != nil {
		return http.StatusBadRequest, err
	}
	oldObject, err := json.Marshal(map[string]any{
		"name":    data.FullName,
		"email":   data.Email,
		"status":  data.IsActive,
		"phone":   data.Phone,
		"role_id": data.RoleId,
	})
	if err != nil {
		return http.StatusBadRequest, err
	}
	paramsLogHistory = model.UserLogHistory{
		UserId:    userAuth.ID,
		ObjectId:  id,
		Action:    "delete",
		Model:     LogHistoryModelName,
		NewObject: nil,
		OldObject: oldObject,
	}
	err = s.Repository.CreateUserLogHistory(paramsLogHistory)
	if err != nil {
		return http.StatusBadRequest, err
	}
	return http.StatusOK, nil
}

func (s *userService) ResendEmailVerification(email string) (int, error) {
	data, err := s.Repository.QuerySelectUser([]string{"musers.id, musers.is_active"}, &model.User{Email: email})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return http.StatusBadRequest, gorm.ErrRecordNotFound
		}
		return http.StatusInternalServerError, err
	}
	if data.IsActive != 2 {
		return http.StatusBadRequest, errors.New("user has been activated")
	}

	rand.Seed(time.Now().UnixNano())
	otp := strconv.Itoa(rand.Int())[:6]
	claims := model.UserActivationClaims{
		Id:             data.Id,
		Otp:            otp,
		StandardClaims: &jwt.StandardClaims{},
	}
	token, err := encrypt.NewWithClaims(claims)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if err = s.Repository.QueryUpdateValuesById(&data.Id, map[string]interface{}{
		"reset_password_token": token,
		"reset_password_otp":   otp,
	}); err != nil {
		return http.StatusBadRequest, err
	}

	body := template.UserVerifyAccount
	body = strings.ReplaceAll(body, "{{replace.otp}}", otp)
	body = strings.ReplaceAll(body, "{{replace.url}}", env.String("BASE_URL", "")+"/email-verification?activation_token="+token)
	msg := smtp.SendEmailSmtpRequest{
		To:      []string{email},
		Subject: "Account Activation - CBQA Global",
		Body:    body,
	}
	resCode, err := smtp.SendEmailSmtp(msg)
	if err != nil {
		return resCode, err
	}
	return http.StatusOK, nil
}

func (s *userService) ValidateEmail(email string) (int, error) {
	data, err := s.Repository.ValidateEmail(email)
	if err != nil {
		return data, nil
	}
	return data, nil
}

func sendVerificationEmailAsync(msg smtp.SendEmailSmtpRequest) {
	go func(emailMsg smtp.SendEmailSmtpRequest) {
		if _, err := smtp.SendEmailSmtp(emailMsg); err != nil {
			log.Printf("failed to send verification email to %v: %v", emailMsg.To, err)
		}
	}(msg)
}

// parseAndCapitalizeName splits full name into first and last token.
func parseAndCapitalizeName(fullName string) (string, string) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return "", ""
	}

	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "", ""
	}

	firstName := strings.ToUpper(parts[0])
	lastName := ""
	if len(parts) > 1 {
		lastName = strings.ToUpper(parts[len(parts)-1])
	}

	return firstName, lastName
}

// capitalizeFullName normalizes spaces and converts full name to uppercase.
func capitalizeFullName(fullName string) string {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return ""
	}
	return strings.ToUpper(strings.Join(strings.Fields(fullName), " "))
}
