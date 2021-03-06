package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

type UserAccount struct {
	Id       int    `json:"id" bson:"id"`
	Username string `json:"username" bson:"username"`
	Name     string `json:"name" bson:"name"`
	Pin      int    `json:"pin" bson:"pin"`
}

type BankAccount struct {
	Id    int    `json:"id" bson:"id"`
	Name  string `json:"name" bson:"name"`
	Owner int    `json:"ownerid" bson:"ownerid"`
}

type Bucket struct {
	Id    int    `json:"id" bson:"id"`
	Name  string `json:"name" bson:"name"`
	Owner int    `json:"ownerid" bson:"ownerid"`
}

type LineItem struct {
	Id          int     `json:"id" bson:"id"`
	Title       string  `json:"title" bson:"title"`
	Description string  `json:"description" bson:"description"`
	Amount      float64 `json:"amount" bson:"amount"`
	Bucket      int     `json:"bucket" bson:"bucket"`
	Bank        int     `json:"bank" bson:"bank"`
	Owner       int     `json:"ownerid" bson:"ownerid"`
}

const (
	DB_USER     = "admin"
	DB_PASSWORD = "admin"
	DB_NAME     = "goproject"
)

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

func db_init() *sql.DB {
	conn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", "db", 5432, DB_USER, DB_PASSWORD, DB_NAME)
	db, err := sql.Open("postgres", conn)

	if err != nil {
		log.Fatal("Failed to connect to the database.")
	}

	fmt.Println("Connected to Database!")

	return db
}

func init() {
	file, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	InfoLogger.Println("Starting the application...")
	http.ListenAndServe(":9000", handler())
}

func checkError(err error) {
	if err != nil {
		ErrorLogger.Println("Internal Error Occured. " + err.Error())
	}
}

func handler() http.HandlerFunc {
	InfoLogger.Println("Handler Listening at :9000 ...")
	return func(w http.ResponseWriter, r *http.Request) {
		var id int
		if r.URL.Path == "/users" {
			userProcess(w, r)
		} else if r.URL.Path == "/banks" {
			bankProcess(w, r)
		} else if r.URL.Path == "/buckets" {
			bucketProcess(w, r)
		} else if r.URL.Path == "/lineitems" {
			lineitemProcess(w, r)
		} else if r.URL.Path == "/authorize" {
			authorize(w, r)
		} else if n, _ := fmt.Sscanf(r.URL.Path, "/user/%d", &id); n == 1 {
			userProcessId(id, w, r)
		} else if n, _ := fmt.Sscanf(r.URL.Path, "/bank/%d", &id); n == 1 {
			bankProcessId(id, w, r)
		} else if n, _ := fmt.Sscanf(r.URL.Path, "/bucket/%d", &id); n == 1 {
			bucketProcessId(id, w, r)
		} else if n, _ := fmt.Sscanf(r.URL.Path, "/lineitem/%d", &id); n == 1 {
			lineitemProcessId(id, w, r)
		}
	}
}

func userProcess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "POST":
		db := db_init()
		defer db.Close()

		var user UserAccount
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var newUserId int
		err := db.QueryRow(
			"INSERT INTO public.useraccount (username, \"name\", pin) VALUES($1, $2, $3) RETURNING id;",
			user.Username,
			user.Name,
			user.Pin,
		).Scan(&newUserId)

		InfoLogger.Println("New User Created.")

		//checkError(err)
		if err != nil {
			res := strings.Contains(string(err.Error()), "duplicate key value violates unique constraint")

			if res {
				http.Error(w, "Username already in use.", http.StatusForbidden)
				ErrorLogger.Println("Failed to create new user. Username in use.")
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}

		user.Id = newUserId

		if err := json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}

	case "GET":
		db := db_init()
		defer db.Close()

		rows, err := db.Query("SELECT id, username, \"name\", pin FROM public.useraccount;")

		checkError(err)

		var users []UserAccount
		for rows.Next() {
			var id, pin int
			var username, name string

			err = rows.Scan(&id, &username, &name, &pin)
			checkError(err)

			users = append(users, UserAccount{
				Id:       id,
				Username: username,
				Name:     name,
				Pin:      pin,
			})
		}
		InfoLogger.Println("Retrieved User Account List.")
		if err := json.NewEncoder(w).Encode(users); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}

	case "PUT":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring request...")
		return
	case "DELETE":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring request...")
		return
	}
}

func userProcessId(id int, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		db := db_init()
		defer db.Close()

		rows, err := db.Query("SELECT id, username, \"name\", pin FROM public.useraccount WHERE id=$1;", id)

		checkError(err)

		var users []UserAccount
		for rows.Next() {
			var id, pin int
			var username, name string

			err = rows.Scan(&id, &username, &name, &pin)
			checkError(err)

			users = append(users, UserAccount{
				Id:       id,
				Username: username,
				Name:     name,
				Pin:      pin,
			})
		}
		InfoLogger.Println("Retrieved Information on specific user.")
		if users == nil || len(users) < 1 {
			http.Error(w, "Not Found!", http.StatusNotFound)
			ErrorLogger.Println("User requested not found.")
			return
		}

		if err := json.NewEncoder(w).Encode(users); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "POST":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring request...")
		return
	case "PUT":
		db := db_init()
		defer db.Close()

		var user UserAccount
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}

		var updatedId int
		err := db.QueryRow(
			"UPDATE public.useraccount SET username=$1, \"name\"=$2, pin=$3 WHERE id=$4 RETURNING id;",
			user.Username,
			user.Name,
			user.Pin,
			id,
		).Scan(&updatedId)

		user.Id = id

		checkError(err)
		InfoLogger.Println("Updated Information of a specific user.")

		if err := json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "DELETE":
		db := db_init()
		defer db.Close()

		var user UserAccount
		err := db.QueryRow("DELETE FROM public.useraccount where id = $1 RETURNING id, username, \"name\", pin;", id).Scan(
			&user.Id,
			&user.Username,
			&user.Name,
			&user.Pin,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
		InfoLogger.Println("User deleted from system.")

		if err := json.NewEncoder(w).Encode(user); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	}
}

func bankProcess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "POST":
		db := db_init()
		defer db.Close()

		var bank BankAccount
		if err := json.NewDecoder(r.Body).Decode(&bank); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}

		var newBankId int
		err := db.QueryRow(
			"INSERT INTO public.bankaccount (\"name\", ownerid) VALUES($1, $2) RETURNING id;",
			bank.Name,
			bank.Owner,
		).Scan(&newBankId)

		checkError(err)
		InfoLogger.Println("New Bank Created.")

		bank.Id = newBankId

		if err := json.NewEncoder(w).Encode(bank); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "GET":
		db := db_init()
		defer db.Close()

		rows, err := db.Query("SELECT id, \"name\", ownerid FROM public.bankaccount;")

		checkError(err)

		var accounts []BankAccount
		for rows.Next() {
			var id, owner int
			var name string

			err = rows.Scan(&id, &name, &owner)
			checkError(err)

			accounts = append(accounts, BankAccount{
				Id:    id,
				Name:  name,
				Owner: owner,
			})
		}
		InfoLogger.Println("Bank Information Retrieved.")
		if err := json.NewEncoder(w).Encode(accounts); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "PUT":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	case "DELETE":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	}
}

func bankProcessId(id int, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		db := db_init()
		defer db.Close()

		rows, err := db.Query("SELECT id, \"name\", ownerid FROM public.bankaccount WHERE id=$1;", id)

		checkError(err)

		var banks []BankAccount
		for rows.Next() {
			var id, ownerid int
			var name string

			err = rows.Scan(&id, &name, &ownerid)
			checkError(err)

			banks = append(banks, BankAccount{
				Id:    id,
				Name:  name,
				Owner: ownerid,
			})
		}
		InfoLogger.Println("Bank Information Retrieved.")
		if banks == nil || len(banks) < 1 {
			http.Error(w, "Not Found!", http.StatusNotFound)
			ErrorLogger.Println("Bank Information Empty/Not Found.")
			return
		}

		if err := json.NewEncoder(w).Encode(banks); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "POST":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	case "PUT":
		db := db_init()
		defer db.Close()

		var bank BankAccount
		if err := json.NewDecoder(r.Body).Decode(&bank); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var updatedId int
		err := db.QueryRow(
			"UPDATE public.bankaccount SET \"name\"=$1 WHERE id=$2 RETURNING id;",
			bank.Name,
			id,
		).Scan(&updatedId)

		checkError(err)

		bank.Id = id
		InfoLogger.Println("Bank Information Updated.")

		if err := json.NewEncoder(w).Encode(bank); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "DELETE":
		db := db_init()
		defer db.Close()

		var bank BankAccount
		err := db.QueryRow("DELETE FROM public.bankaccount where id = $1 RETURNING id,\"name\", ownerid;", id).Scan(
			&bank.Id,
			&bank.Name,
			&bank.Owner,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
		InfoLogger.Println("Bank Information Deleted.")

		if err := json.NewEncoder(w).Encode(bank); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	}
}

func bucketProcess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "POST":
		db := db_init()
		defer db.Close()

		var bucket Bucket
		if err := json.NewDecoder(r.Body).Decode(&bucket); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}

		var newBucketId int
		err := db.QueryRow(
			"INSERT INTO public.bucket (\"name\", ownerid) VALUES($1, $2) RETURNING id;",
			bucket.Name,
			bucket.Owner,
		).Scan(&newBucketId)

		checkError(err)
		InfoLogger.Println("New Bucket Created.")

		bucket.Id = newBucketId

		if err := json.NewEncoder(w).Encode(bucket); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "GET":
		db := db_init()
		defer db.Close()

		rows, err := db.Query("SELECT id, \"name\", ownerid FROM public.bucket;")

		checkError(err)

		var buckets []Bucket
		for rows.Next() {
			var id, owner int
			var name string

			err = rows.Scan(&id, &name, &owner)
			checkError(err)

			buckets = append(buckets, Bucket{
				Id:    id,
				Name:  name,
				Owner: owner,
			})
		}
		InfoLogger.Println("Bucket Information retrieved.")
		if err := json.NewEncoder(w).Encode(buckets); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "PUT":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	case "DELETE":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	}
}

func bucketProcessId(id int, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		db := db_init()
		defer db.Close()

		rows, err := db.Query("SELECT id, \"name\", ownerid FROM public.bucket WHERE id=$1;", id)

		checkError(err)

		var buckets []Bucket
		for rows.Next() {
			var id, ownerid int
			var name string

			err = rows.Scan(&id, &name, &ownerid)
			checkError(err)

			buckets = append(buckets, Bucket{
				Id:    id,
				Name:  name,
				Owner: ownerid,
			})
		}
		InfoLogger.Println("Bucket Information retrieved.")

		if buckets == nil || len(buckets) < 1 {
			http.Error(w, "Not Found!", http.StatusNotFound)
			ErrorLogger.Println("Bucket Information Empty/Not Found.")
			return
		}

		if err := json.NewEncoder(w).Encode(buckets); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "POST":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	case "PUT":
		db := db_init()
		defer db.Close()

		var bucket Bucket
		if err := json.NewDecoder(r.Body).Decode(&bucket); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var updatedId int
		err := db.QueryRow(
			"UPDATE public.bucket SET \"name\"=$1 WHERE id=$2 RETURNING id;",
			bucket.Name,
			id,
		).Scan(&updatedId)

		checkError(err)
		InfoLogger.Println("Bucket Information Updated.")
		bucket.Id = id

		if err := json.NewEncoder(w).Encode(bucket); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "DELETE":
		db := db_init()
		defer db.Close()

		var bucket Bucket
		err := db.QueryRow("DELETE FROM public.bucket where id = $1 RETURNING id,\"name\", ownerid;", id).Scan(
			&bucket.Id,
			&bucket.Name,
			&bucket.Owner,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
		InfoLogger.Println("Bucket Information deleted.")
		if err := json.NewEncoder(w).Encode(bucket); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	}
}

func lineitemProcess(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "POST":
		db := db_init()
		defer db.Close()

		var lineitem LineItem
		if err := json.NewDecoder(r.Body).Decode(&lineitem); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var newLineItemId int
		err := db.QueryRow(
			"INSERT INTO public.lineitem (title, description, amount, bucket, bank, ownerid) VALUES($1, $2, $3, $4, $5, $6) RETURNING id;",
			lineitem.Title,
			lineitem.Description,
			lineitem.Amount,
			lineitem.Bucket,
			lineitem.Bank,
			lineitem.Owner,
		).Scan(&newLineItemId)

		checkError(err)
		InfoLogger.Println("New Line Item Entry created.")

		lineitem.Id = newLineItemId

		if err := json.NewEncoder(w).Encode(lineitem); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}

	case "GET":
		db := db_init()
		defer db.Close()

		rows, err := db.Query("SELECT id, title, description, amount, bucket, bank, ownerid FROM public.lineitem;")

		checkError(err)

		var lineitems []LineItem
		for rows.Next() {
			var id, bucket, bank, ownerid int
			var title, description string
			var amount float64

			err = rows.Scan(&id, &title, &description, &amount, &bucket, &bank, &ownerid)
			checkError(err)

			lineitems = append(lineitems, LineItem{
				Id:          id,
				Title:       title,
				Description: description,
				Amount:      amount,
				Bucket:      bucket,
				Bank:        bank,
				Owner:       ownerid,
			})
		}
		InfoLogger.Println("Line Item Entries retrieved.")

		if err := json.NewEncoder(w).Encode(lineitems); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "PUT":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	case "DELETE":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	}
}

func lineitemProcessId(id int, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		db := db_init()
		defer db.Close()

		rows, err := db.Query("SELECT id, title, description, amount, bucket, bank, ownerid FROM public.lineitem WHERE id=$1;", id)

		checkError(err)

		var lineitems []LineItem
		for rows.Next() {
			var id, bucket, bank, ownerid int
			var amount float64
			var title, description string

			err = rows.Scan(&id, &title, &description, &amount, &bucket, &bank, &ownerid)
			checkError(err)

			lineitems = append(lineitems, LineItem{
				Id:          id,
				Title:       title,
				Description: description,
				Amount:      amount,
				Bucket:      bucket,
				Bank:        bank,
				Owner:       ownerid,
			})
		}

		if lineitems == nil || len(lineitems) < 1 {
			http.Error(w, "Not Found!", http.StatusNotFound)
			ErrorLogger.Println("Line Item Information Empty/Not Found.")
			return
		}

		InfoLogger.Println("Line Item Entry Information retrieved.")

		if err := json.NewEncoder(w).Encode(lineitems); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "POST":
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring Request.")
		return
	case "PUT":
		db := db_init()
		defer db.Close()

		var lineitem LineItem
		if err := json.NewDecoder(r.Body).Decode(&lineitem); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var updatedId int
		err := db.QueryRow(
			"UPDATE public.lineitem SET title=$1, description=$2, amount=$3, bucket=$4, bank=$5 WHERE id=$6 RETURNING id;",
			lineitem.Title,
			lineitem.Description,
			lineitem.Amount,
			lineitem.Bucket,
			lineitem.Bank,
			id,
		).Scan(&updatedId)

		checkError(err)
		InfoLogger.Println("Line Item Entry Information Updated.")

		lineitem.Id = id

		if err := json.NewEncoder(w).Encode(lineitem); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	case "DELETE":
		db := db_init()
		defer db.Close()

		var lineitem LineItem
		err := db.QueryRow("DELETE FROM public.lineitem where id = $1 RETURNING id, title, description, amount, bucket, bank, ownerid;", id).Scan(
			&lineitem.Id,
			&lineitem.Title,
			&lineitem.Description,
			&lineitem.Amount,
			&lineitem.Bucket,
			&lineitem.Bank,
			&lineitem.Owner,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
		InfoLogger.Println("Line Item Entry Information deleted.")

		if err := json.NewEncoder(w).Encode(lineitem); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	}
}

type Login struct {
	Username string `json:"username"`
	Pin      int    `json:"pin"`
}

func authorize(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "POST":
		db := db_init()
		defer db.Close()

		var login Login

		if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
		var users []UserAccount

		rows, err := db.Query("SELECT id, username, \"name\", pin FROM public.useraccount WHERE username=$1 and pin=$2 LIMIT 1;", login.Username, login.Pin)
		checkError(err)

		for rows.Next() {
			var id, pin int
			var username, name string

			err = rows.Scan(&id, &username, &name, &pin)
			checkError(err)

			users = append(users, UserAccount{
				Id:       id,
				Username: username,
				Name:     name,
				Pin:      pin,
			})
		}
		InfoLogger.Println("Authorization request received.")

		if len(users) == 0 {
			http.Error(w, "Not Found", http.StatusNotFound)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}

		if err := json.NewEncoder(w).Encode(users[0]); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			ErrorLogger.Println("Internal Error Occured. " + err.Error())
			return
		}
	default:
		http.Error(w, "Not allowed!", http.StatusMethodNotAllowed)
		WarningLogger.Println("Invalid Operation Requested. Ignoring request...")
		return
	}
}
