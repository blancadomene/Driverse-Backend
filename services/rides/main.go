package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"
)

type Ride struct {
	ID                  string `json:"id"`
	Driver              string `json:"driver"`
	StartDate           string `json:"startDate"`
	EndDate             string `json:"endDate"`
	DeparturePoint      string `json:"departurePoint"`
	DepartureLatLng     string `json:"departureLatLng"`
	DepartureHour       string `json:"departureHour"`
	ArrivalPoint        string `json:"arrivalPoint"`
	ArrivalLatLng       string `json:"arrivalLatLng"`
	ArrivalHour         string `json:"arrivalHour"`
	AvailableSeats      string `json:"availableSeats"`
	PricePerSeat        string `json:"pricePerSeat"`
	AvailableDaysOfWeek int    `json:"availableDaysOfWeek"`
}

var (
	database *sql.DB

	matchingRidesRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "matching_rides_requests",
		Help: "Matching rides processed requests.",
	})

	getRideRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "get_ride_requests",
		Help: "Get ride info processed requests.",
	})

	postRideRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "post_ride_requests",
		Help: "Post ride info processed requests.",
	})

	postBookingRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "post_booking_requests",
		Help: "Post booking processed requests.",
	})

	getBookingRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "get_booking_requests",
		Help: "Get booking processed requests.",
	})
)

func getMatchingRidesInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	matchingRidesRequests.Inc()

	start := r.FormValue("StartDate")
	end := r.FormValue("EndDate")
	depLatLng := r.FormValue("DepartureLatLng")
	depLatLngSplit := strings.Split(depLatLng, ",")
	departurePointRadius := r.FormValue("DeparturePointRadius")
	arrLatLng := r.FormValue("ArrivalLatLng")
	arrLatLngSplit := strings.Split(arrLatLng, ",")
	arrivalPointRadius := r.FormValue("ArrivalPointRadius")
	depHour := r.FormValue("DepartureHour")
	numbSeats := r.FormValue("NumberOfSeats")
	avWeek := r.FormValue("AvailableDaysOfWeek")

	// FUTURE WORK: Use function for distance: https://stackoverflow.com/a/48263512/16030066
	query := `
		SELECT *
		FROM rides
		WHERE (
			StartDate = STR_TO_DATE(?, ?) AND
			EndDate = STR_TO_DATE(?, ?) AND 
			111.111 * DEGREES(
				ACOS(
					LEAST(1.0, 
						COS(RADIANS(DepartureLat))
						* COS(RADIANS(?))
						* COS(RADIANS(DepartureLng - ?))
						+ SIN(RADIANS(DepartureLat))
						* SIN(RADIANS(?))
					)
				)
			) * 1000 <= ? AND
			111.111 * DEGREES(
				ACOS(
					LEAST(1.0, 
						COS(RADIANS(ArrivalLat))
						* COS(RADIANS(?))
						* COS(RADIANS(ArrivalLng - ?))
						+ SIN(RADIANS(ArrivalLat))
						* SIN(RADIANS(?))
					)
				)
			) * 1000 <= ? AND 
			DepartureHour = ? AND 
			AvailableSeats >= ? AND
			AvailableDaysOfWeek & ? = ?
		)`

	results, err := database.Query(query,
		start, "%d/%m/%Y",
		end, "%d/%m/%Y",
		depLatLngSplit[0], depLatLngSplit[1], depLatLngSplit[0],
		departurePointRadius,
		arrLatLngSplit[0], arrLatLngSplit[1], arrLatLngSplit[0],
		arrivalPointRadius,
		depHour,
		numbSeats,
		avWeek, avWeek)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	var rides []Ride

	for results.Next() {

		var ride Ride
		var departureLat, departureLng float32
		var arrivalLat, arrivalLng float32

		err = results.Scan(&ride.ID, &ride.Driver, &ride.StartDate, &ride.EndDate, &ride.DeparturePoint, &departureLat, &departureLng, &ride.DepartureHour, &ride.ArrivalPoint, &arrivalLat, &arrivalLng, &ride.ArrivalHour, &ride.AvailableSeats, &ride.PricePerSeat, &ride.AvailableDaysOfWeek)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error(err)
			return
		}

		ride.DepartureLatLng = fmt.Sprintf("%f,%f", departureLat, departureLng)
		ride.ArrivalLatLng = fmt.Sprintf("%f,%f", arrivalLat, arrivalLng)

		rides = append(rides, ride)
	}

	err = json.NewEncoder(w).Encode(rides)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getRideInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	getRideRequests.Inc()

	rideID := r.FormValue("ID")

	query := "SELECT * FROM rides WHERE (ID = ?)"
	results, err := database.Query(query, rideID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	if !results.Next() {
		w.WriteHeader(http.StatusNotFound)
		log.Error(err)
		return
	}

	var ride Ride
	var departureLat, departureLng float32
	var arrivalLat, arrivalLng float32
	err = results.Scan(&ride.ID, &ride.Driver, &ride.StartDate, &ride.EndDate, &ride.DeparturePoint, &departureLat, &departureLng, &ride.DepartureHour, &ride.ArrivalPoint, &arrivalLat, &arrivalLng, &ride.ArrivalHour, &ride.AvailableSeats, &ride.PricePerSeat, &ride.AvailableDaysOfWeek)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	ride.DepartureLatLng = fmt.Sprintf("%f,%f", departureLat, departureLng)
	ride.ArrivalLatLng = fmt.Sprintf("%f,%f", arrivalLat, arrivalLng)

	err = json.NewEncoder(w).Encode(ride)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func postRideInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	postRideRequests.Inc()

	var ride Ride
	err := json.NewDecoder(r.Body).Decode(&ride)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	DepartureLatLngSplit := strings.Split(ride.DepartureLatLng, ",")

	ArrivalLatLngSplit := strings.Split(ride.ArrivalLatLng, ",")

	query := `
	INSERT INTO rides
	VALUES (
		?,
		?,
		STR_TO_DATE(?, ?),
		STR_TO_DATE(?, ?),
		?,
		?, ?,
		?,
		?,
		?, ?,
		?,
		?,
		?,
		?
	)`

	insert, err := database.Query(query,
		ride.ID,
		ride.Driver,
		ride.StartDate, "%d/%m/%Y",
		ride.EndDate, "%d/%m/%Y",
		ride.DeparturePoint,
		DepartureLatLngSplit[0], DepartureLatLngSplit[1],
		ride.DepartureHour,
		ride.ArrivalPoint,
		ArrivalLatLngSplit[0], ArrivalLatLngSplit[1],
		ride.ArrivalHour,
		ride.AvailableSeats,
		ride.PricePerSeat,
		ride.AvailableDaysOfWeek)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}
	_ = insert
	insert.Close()
	w.WriteHeader(http.StatusOK)
}

func postBooking(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	postBookingRequests.Inc()

	type bookingInfo struct {
		UserID string `json:"userID"`
		RideID string `json:"rideID"`
		Seats  int    `json:"seats"`
	}

	var info bookingInfo
	err := json.NewDecoder(r.Body).Decode(&info)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	for i := 0; i < info.Seats; i++ {
		query := "INSERT INTO bookings VALUES (?, ?)"
		insert, err := database.Query(query, info.UserID, info.RideID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error(err)
			return
		}
		_ = insert
		insert.Close()
	}
	w.WriteHeader(http.StatusOK)
}

func getBooking(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	getBookingRequests.Inc()

	userID := r.FormValue("ID")

	query := `
		SELECT rides.*
		FROM rides
		INNER JOIN bookings ON bookings.rideID=rides.ID
		WHERE (bookings.userID = ?)`

	results, err := database.Query(query, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

	var rides []Ride

	for results.Next() {

		var ride Ride
		var departureLat, departureLng float32
		var arrivalLat, arrivalLng float32

		err = results.Scan(&ride.ID, &ride.Driver, &ride.StartDate, &ride.EndDate, &ride.DeparturePoint, &departureLat, &departureLng, &ride.DepartureHour, &ride.ArrivalPoint, &arrivalLat, &arrivalLng, &ride.ArrivalHour, &ride.AvailableSeats, &ride.PricePerSeat, &ride.AvailableDaysOfWeek)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error(err)
			return
		}

		ride.DepartureLatLng = fmt.Sprintf("%f,%f", departureLat, departureLng)
		ride.ArrivalLatLng = fmt.Sprintf("%f,%f", arrivalLat, arrivalLng)

		rides = append(rides, ride)
	}

	err = json.NewEncoder(w).Encode(rides)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Error(err)
		return
	}

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
		Methods(http.MethodGet).
		Path("/info").
		Queries(
			"StartDate", "{StartDate}",
			"EndDate", "{EndDate}",
			"DepartureLatLng", "{DepartureLatLng}",
			"DeparturePointRadius", "{DeparturePointRadius}",
			"ArrivalLatLng", "{ArrivalLatLng}",
			"ArrivalPointRadius", "{ArrivalPointRadius}",
			"DepartureHour", "{DepartureHour}",
			"NumberOfSeats", "{NumberOfSeats}",
			"AvailableDaysOfWeek", "{AvailableDaysOfWeek}",
		).
		HandlerFunc(getMatchingRidesInfo)

	router.
		Methods(http.MethodPost).
		Path("/info").
		HandlerFunc(postRideInfo)

	router.
		Methods(http.MethodGet).
		Path("/info").
		Queries("ID", "{ID}").
		HandlerFunc(getRideInfo)

	router.
		Methods(http.MethodPost).
		Path("/book").
		HandlerFunc(postBooking)

	router.
		Methods(http.MethodGet).
		Path("/book").
		Queries("ID", "{ID}").
		HandlerFunc(getBooking)

	router.Handle("/metrics", promhttp.Handler())

	fmt.Println("Listening at 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
