package user

import (
	"encoding/json"
	"errors"
	"local/gintest/services/db"
	"local/gintest/services/dbheap"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type userRegisterStruct struct {
	UserName string `json:"Username"`
	Password string `json:"Password"`
}

func (rs *userRegisterStruct) validate() error {
	if len(rs.Password) == 0 {
		return errors.New("The password field is required")
	}
	if len(rs.UserName) == 0 {
		return errors.New("The User Name field is required")
	}
	return nil
}

func Register(w http.ResponseWriter, r *http.Request) {
	buffer := make([]byte, 10)
	bodyData := []byte{}
	for {
		c, err := r.Body.Read(buffer)
		bodyData = append(bodyData, buffer[:c]...)
		if err != nil {
			break
		}
	}
	header := w.Header()
	header.Add("ASD", "EFG")
	log.Println("Info read from the body: ", string(bodyData))
	if !json.Valid(bodyData) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userData := userRegisterStruct{}
	err := json.Unmarshal(bodyData, &userData)
	if err != nil || userData.validate() != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println("User: ", userData.UserName)
	log.Println("Password: ", userData.Password)

	hash, herr := bcrypt.GenerateFromPassword([]byte(userData.Password), bcrypt.DefaultCost)
	if herr != nil {
		log.Println("Error generating hash: ", herr)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dbUser := db.DBUser{
		Username:       userData.UserName,
		HashedPassword: string(hash),
	}
	session, err := dbheap.GetSession()
	if err != nil {
		log.Println("Error getting session: ", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer session.Close()
	err = session.ClientSession.InsertUser(dbUser)
	if err != nil {
		log.Println("Error saving user: ", err)
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(bodyData)
}
