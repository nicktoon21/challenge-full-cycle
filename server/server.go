package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type ResponseAPI struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}

type ResponseCotacao struct {
	Cotacao string `json:"cotacao"`
}

func getDolar() (*ResponseAPI, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*200)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result ResponseAPI
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func getCotacao(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/")

	//Busca a cotacao do Dolar atual
	cotacao, err := getDolar()
	if err != nil {
		panic(err)
	}

	//insere no banco de dados
	err = insertdb(cotacao)
	if err != nil {
		panic(err)
	}

	//Monta o resposta da requisicao
	response := ResponseCotacao{cotacao.USDBRL.Bid}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	//Escreve a resposta da requisicao
	w.Write([]byte(jsonResponse))
}

func insertdb(data *ResponseAPI) error {

	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare(`INSERT INTO cotacoes (cotacao) VALUES (?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*10)
	defer cancel()

	_, err = stmt.ExecContext(ctx, string(data.USDBRL.Bid))
	if err != nil {
		log.Printf("Error: %v", err)
		return err
	}

	return nil

}

func main() {
	var f *os.File
	_, err := os.Stat("./database.db")
	if !os.IsNotExist(err) {
		f, err = os.Create("database.db")
		if err != nil {
			log.Fatal(err.Error())
		}
		defer f.Close()
		db, err := sql.Open("sqlite3", "./database.db")
		if err != nil {
			log.Fatal(err.Error())
		}
		defer db.Close()

		stmt, err := db.Prepare(`
	CREATE TABLE cotacoes(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cotacao VARCHAR(10)
	);
	`)
		if err != nil {
			log.Fatal(err.Error())
		}
		defer stmt.Close()
		_, err = stmt.Exec()
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	http.HandleFunc("/cotacao", getCotacao)
	http.ListenAndServe(":8080", nil)
}
