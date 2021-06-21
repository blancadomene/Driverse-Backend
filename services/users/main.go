package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gorilla/mux"
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
		log.Fatal(err)
	}

	query := fmt.Sprintf("SELECT ID FROM users WHERE (Email = \"%s\" AND Password = \"%s\")", info.Email, info.Password)
	results, err := database.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	if results.Next() {
		w.WriteHeader(http.StatusOK)
		var user User
		err = results.Scan(&user.ID)
		if err != nil {
			log.Fatal(err)
		}

		err := json.NewEncoder(w).Encode(user)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func getUserInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	getUserRequests.Inc()

	id := r.FormValue("ID")

	query := fmt.Sprintf(`
		SELECT *
		FROM users
		WHERE (ID = "%s")`,
		id)
	results, err := database.Query(query)
	if err != nil {
		log.Fatal(err)
	}

	for results.Next() {
		var user User
		// for each row, scan the result into our tag composite object
		err = results.Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.Surname, &user.Birthdate, &user.Car, &user.Image, &user.Mobilephone, &user.Preferences)
		if err != nil {
			log.Fatal(err)
		}
		user.Password = ""

		err := json.NewEncoder(w).Encode(user)
		if err != nil {
			log.Fatal(err)
		}
		break
	}
}

// Future work: Sign up
// Users added from Postman or SQL until then
func postUserInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	postUserRequests.Inc()

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		log.Fatal(err)
	}

	var emailmd5 = md5.Sum([]byte(user.Email))
	var passwordmd5 = md5.Sum([]byte(user.Password))

	var grav = fmt.Sprintf("https://www.gravatar.com/avatar/%s?s=256&d=identicon&r=PG", hex.EncodeToString(emailmd5[:]))

	query := fmt.Sprintf(`
		INSERT INTO users
		VALUES (
			"%s", 
			"%s", 
			"%s", 
			"%s", 
			"%s", 
			STR_TO_DATE("%s", "%s"), 
			"%s", 
			"%s", 
			"%s", 
			"%s"
		)`,
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

	insert, err := database.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	_ = insert
	defer insert.Close()
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
