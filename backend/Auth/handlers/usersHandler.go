package handlers

import (
	"12factorapp/data"
	"12factorapp/db"
	"context"
	"encoding/json"
	"github.com/golang-jwt/jwt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"time"
)

type KeyUser struct{}

type UsersHandler struct {
	logger   *log.Logger
	userRepo db.UserRepo
}

type LogUser struct {
	Username string
	Password string
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

var jwtKey = []byte("secret_key")

// NewUsersHandler Injecting the logger makes this code much more testable.
func NewUsersHandler(l *log.Logger, ur db.UserRepo) *UsersHandler {
	return &UsersHandler{l, ur}
}

func (u *UsersHandler) GetUsers(rw http.ResponseWriter, h *http.Request) {
	users := u.userRepo.GetUsers()
	err := users.ToJSON(rw)

	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		u.logger.Println("Unable to convert to json :", err)
		return
	}

	//cookie, err := h.Cookie("token")
	//if err != nil {
	//	if err == http.ErrNoCookie {
	//		rw.WriteHeader(http.StatusUnauthorized)
	//		return
	//	}
	//	rw.WriteHeader(http.StatusBadRequest)
	//	return
	//}
	//tokenStr := cookie.Value
	//claims := &Claims{}
	//
	//tkn, err := jwt.ParseWithClaims(tokenStr, claims,
	//	func(t *jwt.Token) (interface{}, error) {
	//		return jwtKey, nil
	//	})
	//
	//if err != nil {
	//	if err == jwt.ErrSignatureInvalid {
	//		rw.WriteHeader(http.StatusUnauthorized)
	//		return
	//	}
	//	rw.WriteHeader(http.StatusBadRequest)
	//	return
	//
	//}
	//
	//if !tkn.Valid {
	//	rw.WriteHeader(http.StatusUnauthorized)
	//	return
	//}

}

func (u *UsersHandler) GetUser(rw http.ResponseWriter, h *http.Request) {
	vars := mux.Vars(h)
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		u.logger.Println("Unable to convert from ascii to integer - input was :", vars["id"])
		return
	}

	user, er := u.userRepo.GetUser(id)

	if er != nil {
		http.Error(rw, er.Error(), http.StatusNotFound)
		u.logger.Println("Unable to find user.", er)
		return
	}

	err = user.ToJSON(rw)

	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		u.logger.Println("Unable to convert to json :", err)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (u *UsersHandler) LoginUser(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var logged data.User
	err := decoder.Decode(&logged)

	if err != nil {
		http.Error(rw, "Unable to convert to json", http.StatusInternalServerError)
		u.logger.Println("Unable to convert to json :", err)
		return
	}
	u.logger.Println(logged)
	if !isEmpty(logged.Username) && !isEmpty(logged.Password) {

		_, err := u.userRepo.LoginUser(logged.Username, logged.Password)
		if err != nil {
			http.Error(rw, "Unable to login", http.StatusInternalServerError)
			u.logger.Println("Unable to login", err)
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write([]byte("401 - Unauthorized"))
			return
		}

		expirationTime := time.Now().Add(time.Minute * 5)

		claims := &Claims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expirationTime.Unix(),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtKey)

		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		http.SetCookie(rw,
			&http.Cookie{
				Name:    "token",
				Value:   tokenString,
				Expires: expirationTime,
				Path:    "/",
			})

		var userResponse = make(map[string]string)
		userResponse["token"] = tokenString
		jsonUser, err := json.Marshal(userResponse)
		rw.WriteHeader(http.StatusOK)
		rw.Header().Set("Content-Type", "aplication/json")
		rw.Write(jsonUser)
		return
	}

	rw.WriteHeader(http.StatusNotAcceptable)
	rw.Write([]byte("406 - Not acceptable"))
}

func (u *UsersHandler) Register(rw http.ResponseWriter, h *http.Request) {
	user := h.Context().Value(KeyUser{}).(*data.User)
	if user.Username != "" && user.Password != "" && user.Role != "" {
		if user.Role == "regular" || user.Role == "business" {
			_, err := u.userRepo.GetUserByUsername(user.Username)
			if err == nil {
				rw.WriteHeader(http.StatusNotAcceptable)
				return
			}
			if u.userRepo.Register(user) == true {
				rw.WriteHeader(http.StatusAccepted)
				return
			}
		}
	}
	rw.WriteHeader(http.StatusNotAcceptable)
	rw.Write([]byte("406 - Not acceptable"))
}

//Middleware to try and decode the incoming body. When decoded we run the validation on it just to check if everything is okay
//with the model. If anything is wrong we terminate the execution and the code won't even hit the handler functions.
//With a key we bind what we read to the context of the current request. Later we use that key to get to the read value.

func (u *UsersHandler) MiddlewareUsersValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		user := &data.User{}
		err := user.FromJSON(h.Body)
		if err != nil {
			http.Error(rw, "Unable to decode json", http.StatusBadRequest)
			u.logger.Println(err)
			return
		}

		ctx := context.WithValue(h.Context(), KeyUser{}, user)
		h = h.WithContext(ctx)

		next.ServeHTTP(rw, h)
	})
}

func (u *UsersHandler) MiddlewareContentTypeSet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, h *http.Request) {
		u.logger.Println("Method [", h.Method, "] - Hit path :", h.URL.Path)

		rw.Header().Add("Content-Type", "application/json")

		next.ServeHTTP(rw, h)
	})
}

func isEmpty(data string) bool {
	if len(data) <= 0 {
		return true
	} else {
		return false
	}
}
