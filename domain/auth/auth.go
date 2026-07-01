package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"

	"erp-cbqa-global/domain/auth/model"
	"erp-cbqa-global/domain/auth/repository"
	"erp-cbqa-global/lib/constant"
	"erp-cbqa-global/lib/encrypt"
)

type AuthServiceInterface interface {
	Login(reqBody model.ReqBody) (*model.ResBody, int, error)
	CheckAuth(string) (*model.User, error)
	Logout(model.User) (int, error)
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
	user, err := s.Repository.FirstByUsername(reqBody.Username)
	if err != nil {
		return nil, http.StatusBadRequest, errors.New(constant.UserNotFound)
	}
	if err = encrypt.CompareHashAndPassword(&user.Password, &reqBody.Password); err != nil {
		return nil, http.StatusBadRequest, errors.New(constant.PasswordIsIncorrect)
	}
	claims := model.Jwt{
		ID:       user.ID,
		Username: user.Username,
		RoleID:   user.RoleID,
		StandardClaims: &jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * time.Duration(1)).Unix(),
		},
	}
	ss, err := encrypt.NewWithClaims(&claims)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if err = s.Repository.UpdateToken(user, ss); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	var resBody model.ResBody
	resBody.ID = user.ID
	resBody.Username = user.Username
	resBody.RoleID = user.RoleID
	resBody.Token = ss
	return &resBody, http.StatusOK, nil
}

func (s *authService) CheckAuth(bearerToken string) (*model.User, error) {
	tokenRaw, claims, err := encrypt.Parse(bearerToken)
	if err != nil {
		return nil, err
	}
	id := string(claims["id"].(string))
	user, err := s.Repository.FirstByID(id)
	if err != nil {
		if err.Error() == constant.RecordNotFound {
			return nil, errors.New(constant.UserNotFound)
		}
	}
	if user.Token != tokenRaw {
		return nil, errors.New(constant.UserHasSignedOut)
	}
	return &user, nil
}

func (s *authService) Logout(user model.User) (int, error) {
	if err := s.Repository.DeleteToken(user); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
