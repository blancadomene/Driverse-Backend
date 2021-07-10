package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gorilla/mux"

	log "github.com/Sirupsen/logrus"
)

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	Birthdate   string `json:"birthdate"`
	Car         string `json:"car"`
	Image       string `json:"image"`
	Mobilephone string `json:"mobilephone"`
	Preferences string `json:"preferences"`
}

var (
	database *sql.DB

	userAuthenticationRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "user_authentication_requests",
		Help: "User authentication processed requests.",
	})

	getUserRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "get_user_requests",
		Help: "Get user info processed requests.",
	})

	postUserRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "post_user_requests",
		Help: "Post user info processed requests.",
	})

	userAuthenticationRequestsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "user_authentication_requests_failed",
		Help: "Failed user authentication processed requests.",
	})

	getUserRequestsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "get_user_requests_failed",
		Help: "Failed get user info processed requests.",
	})

	postUserRequestsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "post_user_requests_failed",
		Help: "Failed post user info processed requests.",
	})
)

func Authentication(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	userAuthenticationRequests.Inc()

	type loginInfo struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var info loginInfo
	err := json.NewDecoder(r.Body).Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		userAuthenticationRequestsFailed.Inc()
		log.Error(err)
		return
	}

	query := "SELECT ID FROM users WHERE (Email = ? AND Password = ?)"
	results, err := database.Query(query, info.Email, info.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		userAuthenticationRequestsFailed.Inc()
		log.Error(err)
		return
	}

	if results.Next() {
		var user User
		err = results.Scan(&user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			userAuthenticationRequestsFailed.Inc()
			log.Error(err)
			return
		}

		err := json.NewEncoder(w).Encode(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			userAuthenticationRequestsFailed.Inc()
			log.Error(err)
			return
		}

		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		userAuthenticationRequestsFailed.Inc()
	}
}

func getUserInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	getUserRequests.Inc()

	id := r.FormValue("ID")

	query := "SELECT * FROM users WHERE (ID = ?)"
	results, err := database.Query(query, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		getUserRequestsFailed.Inc()
		log.Error(err)
		return
	}

	for results.Next() {
		var user User
		// for each row, scan the result into our tag composite object
		err = results.Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.Surname, &user.Birthdate, &user.Car, &user.Image, &user.Mobilephone, &user.Preferences)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			getUserRequestsFailed.Inc()
			log.Error(err)
			return
		}
		user.Password = ""

		err := json.NewEncoder(w).Encode(user)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			getUserRequestsFailed.Inc()
			log.Error(err)
			return
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	getUserRequestsFailed.Inc()
}

// Future work: Sign up
// Users added from Postman or SQL until then
func postUserInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	postUserRequests.Inc()

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		postUserRequestsFailed.Inc()
		log.Error(err)
		return
	}

	var emailmd5 = md5.Sum([]byte(user.Email))
	var passwordmd5 = md5.Sum([]byte(user.Password))

	var grav = fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=256&d=identicon&r=PG", hex.EncodeToString(emailmd5[:]))

	query := `
		INSERT INTO users
		VALUES (
			?, 
			?, 
			?, 
			?, 
			?, 
			STR_TO_DATE(?, ?), 
			?, 
			?, 
			?, 
			?
		)`
	insert, err := database.Query(query,
		user.ID,
		user.Email,
		hex.EncodeToString(passwordmd5[:]),
		user.Name,
		user.Surname,
		user.Birthdate, "%d/%m/%Y",
		user.Car,
		grav,
		user.Mobilephone,
		user.Preferences)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		postUserRequestsFailed.Inc()
		log.Error(err)
		return
	}

	_ = insert
	insert.Close()

	w.WriteHeader(http.StatusOK)
}

func main() {
	var err error

	database, err = sql.Open("mysql", "root:secret@tcp(database:3306)/driverse")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	var router = mux.NewRouter()

	router.
		Methods(http.MethodPost).
		Path("/login").
		HandlerFunc(Authentication)

	router.
		Methods(http.MethodPost).
		Path("/info").
		HandlerFunc(postUserInfo)

	router.
		Methods(http.MethodGet).
		Path("/info").
		Queries("ID", "{ID}").
		HandlerFunc(getUserInfo)

	router.Handle("/metrics", promhttp.Handler())

	fmt.Println("Listening at 8080")
	log.Fatal(http.ListenAndServe(":8080", router))

}
