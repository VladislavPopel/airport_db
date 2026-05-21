package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// Конфигурация подключения к  PostgreSQL
const connStr = "host=postgres_db port=5432 user=lab_user password=lab_password dbname=airport_db sslmode=disable"

var db *sql.DB
var tmpl *template.Template

// Структуры данных
type Airport struct {
	ID                  int
	IataCode            string
	Name                string
	City                string
	Country             string
	OpenedDateRaw       string // ГГГГ-ММ-ДД для html-формы
	OpenedDateFormatted string // ДД.ММ.ГГГГ для отображения пользователю
	RunwaysCount        int
	RunwayLengthKm      float64
	AltitudeM           int
}

type Flight struct {
	ID                     int
	FlightNumber           string
	AirportFromID          int
	AirportToID            int
	AirportFromName        string // Из JOIN
	AirportFromCity        string // Из JOIN
	AirportToName          string // Из JOIN
	AirportToCity          string // Из JOIN
	DepartureDateRaw       string
	DepartureDateFormatted string
	DurationMinutes        int
	DistanceKm             float64
	BasePriceUsd           float64
	SeatsTotal             int
}

type PageData struct {
	CurrentTab string
	Airports   []Airport
	Flights    []Flight
}

func main() {
	var err error
	// Подключение к БД
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Не удалось подключиться к базе данных: ", err)
	}

	// Парсинг шаблонов
	tmpl = template.Must(template.ParseFiles("templates.html"))

	// Роутинг
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/save-airport", handleSaveAirport)
	http.HandleFunc("/save-flight", handleSaveFlight)
	http.HandleFunc("/delete", handleDelete)

	fmt.Println("Сервер успешно запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Главная страница
func handleIndex(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "airports" // вкладка по умолчанию
	}

	data := PageData{CurrentTab: tab}

	// Всегда подгружаем аэропорты, так как они нужны и для выпадающих списков во вкладке рейсов
	airports, err := getAllAirports()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data.Airports = airports

	if tab == "flights" {
		flights, err := getAllFlights()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Flights = flights
	}

	tmpl.Execute(w, data)
}

// CRUD: Сохранение / Изменение Аэропорта
func handleSaveAirport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	idStr := r.FormValue("id")
	iata := r.FormValue("iata_code")
	name := r.FormValue("name")
	city := r.FormValue("city")
	country := r.FormValue("country")
	opened := r.FormValue("opened_date")
	runways, _ := strconv.Atoi(r.FormValue("runways_count"))
	length, _ := strconv.ParseFloat(r.FormValue("runway_length_km"), 64)
	altitude, _ := strconv.Atoi(r.FormValue("altitude_m"))

	if idStr == "" {
		// INSERT
		_, err := db.Exec(`INSERT INTO airports (iata_code, name, city, country, opened_date, runways_count, runway_length_km, altitude_m) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, iata, name, city, country, opened, runways, length, altitude)
		if err != nil {
			http.Error(w, "Ошибка добавления: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// UPDATE
		id, _ := strconv.Atoi(idStr)
		_, err := db.Exec(`UPDATE airports SET iata_code=$1, name=$2, city=$3, country=$4, opened_date=$5, runways_count=$6, runway_length_km=$7, altitude_m=$8 WHERE id=$9`,
			iata, name, city, country, opened, runways, length, altitude, id)
		if err != nil {
			http.Error(w, "Ошибка обновления: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/?tab=airports", http.StatusSeeOther)
}

// CRUD: Сохранение / Изменение Рейса
func handleSaveFlight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	idStr := r.FormValue("id")
	flightNum := r.FormValue("flight_number")
	fromID, _ := strconv.Atoi(r.FormValue("airport_from_id"))
	toID, _ := strconv.Atoi(r.FormValue("airport_to_id"))
	departure := r.FormValue("departure_date")
	duration, _ := strconv.Atoi(r.FormValue("duration_minutes"))
	distance, _ := strconv.ParseFloat(r.FormValue("distance_km"), 64)
	price, _ := strconv.ParseFloat(r.FormValue("base_price_usd"), 64)
	seats, _ := strconv.Atoi(r.FormValue("seats_total"))

	if fromID == toID {
		http.Error(w, "Ошибка: Аэропорты отправления и назначения не должны совпадать!", http.StatusBadRequest)
		return
	}

	if idStr == "" {
		// INSERT
		_, err := db.Exec(`INSERT INTO flights (flight_number, airport_from_id, airport_to_id, departure_date, duration_minutes, distance_km, base_price_usd, seats_total) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, flightNum, fromID, toID, departure, duration, distance, price, seats)
		if err != nil {
			http.Error(w, "Ошибка добавления рейса: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// UPDATE
		id, _ := strconv.Atoi(idStr)
		_, err := db.Exec(`UPDATE flights SET flight_number=$1, airport_from_id=$2, airport_to_id=$3, departure_date=$4, duration_minutes=$5, distance_km=$6, base_price_usd=$7, seats_total=$8 WHERE id=$9`,
			flightNum, fromID, toID, departure, duration, distance, price, seats, id)
		if err != nil {
			http.Error(w, "Ошибка обновления рейса: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/?tab=flights", http.StatusSeeOther)
}

// CRUD: Удаление записи
func handleDelete(w http.ResponseWriter, r *http.Request) {
	tab := r.URL.Query().Get("tab")
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)

	if tab == "airports" {
		// Благодарим ON DELETE RESTRICT в структуре БД. Если рейс привязан к аэропорту, запись не удалится.
		_, err := db.Exec("DELETE FROM airports WHERE id = $1", id)
		if err != nil {
			http.Error(w, "Невозможно удалить аэропорт. На него ссылаются существующие рейсы. Сначала удалите соответствующие рейсы.", http.StatusConflict)
			return
		}
	} else if tab == "flights" {
		_, err := db.Exec("DELETE FROM flights WHERE id = $1", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/?tab="+tab, http.StatusSeeOther)
}

// Дополнительные функции выборки из БД
func getAllAirports() ([]Airport, error) {
	rows, err := db.Query("SELECT id, iata_code, name, city, country, opened_date, runways_count, runway_length_km, altitude_m FROM airports ORDER BY iata_code")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Airport
	for rows.Next() {
		var a Airport
		var t time.Time
		err := rows.Scan(&a.ID, &a.IataCode, &a.Name, &a.City, &a.Country, &t, &a.RunwaysCount, &a.RunwayLengthKm, &a.AltitudeM)
		if err != nil {
			return nil, err
		}
		a.OpenedDateRaw = t.Format("2006-01-02")
		a.OpenedDateFormatted = t.Format("02.01.2006") // Перевод в требуемый формат ДД.ММ.ГГГГ
		list = append(list, a)
	}
	return list, nil
}

func getAllFlights() ([]Flight, error) {
	query := `
		SELECT f.id, f.flight_number, f.airport_from_id, f.airport_to_id, 
		       a1.name, a1.city, a2.name, a2.city,
		       f.departure_date, f.duration_minutes, f.distance_km, f.base_price_usd, f.seats_total
		FROM flights f
		JOIN airports a1 ON f.airport_from_id = a1.id
		JOIN airports a2 ON f.airport_to_id = a2.id
		ORDER BY f.flight_number`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Flight
	for rows.Next() {
		var f Flight
		var t time.Time
		err := rows.Scan(&f.ID, &f.FlightNumber, &f.AirportFromID, &f.AirportToID,
			&f.AirportFromName, &f.AirportFromCity, &f.AirportToName, &f.AirportToCity,
			&t, &f.DurationMinutes, &f.DistanceKm, &f.BasePriceUsd, &f.SeatsTotal)
		if err != nil {
			return nil, err
		}
		f.DepartureDateRaw = t.Format("2006-01-02")
		f.DepartureDateFormatted = t.Format("02.01.2006") // Перевод в ДД.ММ.ГГГГ
		list = append(list, f)
	}
	return list, nil
}
