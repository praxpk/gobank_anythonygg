package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-playground/validator"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type APIServer struct {
	listenAddr string
	store      Storage
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type APIError struct {
	Error string `json:"error"`
}

var validate = validator.New()

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
		}
	}
}

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func withJWTAuth(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if len(tokenString) < 7 || strings.ToUpper(tokenString[:7]) != "BEARER "{
			WriteJSON(w, http.StatusForbidden, APIError{Error: "invalid token"})
			return
		}
		token, err := validateJWT(tokenString[7:])
		if err != nil || !token.Valid {
			WriteJSON(w, http.StatusForbidden, APIError{Error: "invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		fmt.Println(claims)
		handlerFunc(w, r)
	}
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expiresAt":     15000,
		"accountId": account.ID,
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}

func validatePassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST"{
		return fmt.Errorf("method not allowed: %s", r.Method)
	}
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}
	if err := validate.Struct(req); err != nil{
		return fmt.Errorf("invalid login request format")
	}
	acc, err := s.store.GetAccountByEmail(req.Email)
	if err!= nil {
		return fmt.Errorf("account does not exist")
	}
	if !validatePassword(req.Password, acc.EncryptedPassword) {
		return fmt.Errorf("incorrect password")
	}
	token, err := createJWT(acc)
	if err!= nil{
		return fmt.Errorf("server error")
	}
	w.Header().Set("Authorization", "Bearer "+token)
	return WriteJSON(w, http.StatusOK, req)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		return s.handleGetAllAccounts(w, r)
	case "POST":
		return s.handleCreateAccount(w, r)
	default:
		return fmt.Errorf("method not allowed: %s", r.Method)
	}
}

func (s *APIServer) handleAccountByID(w http.ResponseWriter, r *http.Request) error {
	id, err := s.getIDFromRequest(r)
	if err != nil {
		return err
	}

	switch r.Method {
	case "GET":
		account, err := s.store.GetAccountByID(id)
		if err != nil {
			return err
		}
		WriteJSON(w, http.StatusOK, &account)

	case "DELETE":
		err = s.store.DeleteAccount(id)
		if err != nil {
			return err
		}
		return WriteJSON(w, http.StatusOK, "OK")

	default:
		return fmt.Errorf("method not allowed: %s", r.Method)
	}
	return nil
}

func (s *APIServer) handleGetAllAccounts(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, accounts)
	return nil
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	createAccountReq := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return err
	}
	if err := validate.Struct(createAccountReq); err != nil{
		return fmt.Errorf("invalid request format")
	}
	existingAccount, _ := s.store.GetAccountByEmail(createAccountReq.Email)  

	if existingAccount != nil {
		return fmt.Errorf("account with email address %s already exists", createAccountReq.Email)
	}

	account, err := NewAccount(createAccountReq.FirstName, createAccountReq.LastName, createAccountReq.Email, createAccountReq.Password)
	if err != nil {
		return err
	}

	if err := s.store.CreateAccount(account); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error {
	tr := new(TransferRequest)
	if err := json.NewDecoder(r.Body).Decode(tr); err != nil {
		return err
	}
	defer r.Body.Close()

	return WriteJSON(w, http.StatusOK, tr)
}

func (s *APIServer) getIDFromRequest(r *http.Request) (int, error) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("id %s provided is not an integer: %v", idStr, err)
	}
	return id, nil
}

func (s *APIServer) Run() {
	router := mux.NewRouter()
	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleAccountByID)))
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransfer))
	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))

	log.Println("JSON API server running on port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}