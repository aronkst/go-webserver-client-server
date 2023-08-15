package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/aronkst/go-webserver-client-server/utils"
	_ "github.com/mattn/go-sqlite3"
)

const API = "https://economia.awesomeapi.com.br/json/last/USD-BRL"

func GetPrice() (utils.Price, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", API, nil)
	if err != nil {
		return utils.Price{}, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return utils.Price{}, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return utils.Price{}, err
	}

	var price utils.Price

	err = json.Unmarshal(body, &price)
	if err != nil {
		return utils.Price{}, err
	}

	return price, nil
}

func SaveToDatabase(db *sql.DB, dollarReal utils.USDBRL) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	stmt, err := db.Prepare(`
	  INSERT INTO USDBRL (code, codein, name, high, low, var_bid, pct_change, bid, ask, timestamp, create_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, dollarReal.Code, dollarReal.Codein, dollarReal.Name, dollarReal.High, dollarReal.Low,
		dollarReal.VarBid, dollarReal.PctChange, dollarReal.Bid, dollarReal.Ask, dollarReal.Timestamp,
		dollarReal.CreateDate)
	if err != nil {
		return err
	}

	return nil
}

func handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cotacao" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		price, err := GetPrice()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		err = SaveToDatabase(db, price.USDBRL)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		bid := utils.Bid{Bid: price.USDBRL.Bid}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(bid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}
}

func main() {
	db, err := sql.Open("sqlite3", "dollar_real.db")
	if err != nil {
		panic(err)
	}

	defer db.Close()

	createTable := `
		CREATE TABLE IF NOT EXISTS USDBRL (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT,
			codein TEXT,
			name TEXT,
			high TEXT,
			low TEXT,
			var_bid TEXT,
			pct_change TEXT,
			bid TEXT,
			ask TEXT,
			timestamp TEXT,
			create_date TEXT
		);
	`
	_, err = db.Exec(createTable)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/cotacao", handler(db))
	http.ListenAndServe(":8080", nil)
}
