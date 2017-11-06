package main

import (
	"database/sql"

	"github.com/linkosmos/mapop"
	"github.com/gorilla/mux"
	"github.com/xeipuuv/gojsonschema"
	_ "github.com/lib/pq"
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"io/ioutil"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

func (a *App) Initialize(user, password, dbname string) {
	connectionString :=
		fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, password, dbname)

	var err error
	a.DB, err = sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	a.Router = mux.NewRouter()
	a.initializeRoutes()
}

func (a *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(":8000", a.Router))
}

func (a *App) initializeRoutes() {
	a.Router.HandleFunc("/schema/{schemaID}", a.createSchema).Methods("POST")
	a.Router.HandleFunc("/schema/{schemaID}", a.getSchema).Methods("GET")
	a.Router.HandleFunc("/validate/{schemaID}", a.validateSchema).Methods("POST")
}

func cleanDocument(document []byte) ([]byte, error) {
	var m map[string]interface{}
	err := json.Unmarshal(document, &m)
	if err != nil {
		panic(err)
	}

	m = mapop.SelectFunc(func(key string, value interface{}) bool {
		return value != nil
	}, m)

	return json.Marshal(&m)
}

func (a *App) createSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["schemaID"]
	s := jsonSchema{SchemaID: id}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if json.Valid(body) {
		s.SchemaDef = string(body)
	} else {
		log.Printf("Invalid json uploaded")
		http.Error(w, "Invalid json uploaded", http.StatusBadRequest)
		return
	}

	if err := s.createSchema(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, s)
}

func (a *App) getSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["schemaID"]
	s := jsonSchema{SchemaID: id}

	if err := s.getSchema(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Schema not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, s)
}

func (a *App) validateSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["schemaID"]
	s := jsonSchema{SchemaID: id}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "can't read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if json.Valid(body) {
		body, err = cleanDocument(body)
		if err != nil {
			log.Printf("Error cleaning document: %v", err)
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		jsonDoc := string(body)

		if err := s.getSchema(a.DB); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		schemaLoader := gojsonschema.NewStringLoader(s.SchemaDef)
		documentLoader := gojsonschema.NewStringLoader(jsonDoc)
		result, err := gojsonschema.Validate(schemaLoader, documentLoader)

		if result.Valid() {
			fmt.Printf("The document is valid\n")
		} else {
			fmt.Printf("The document is not valid. see errors :\n")
			for _, err := range result.Errors() {
				// Err implements the ResultError interface
				fmt.Printf("- %s\n", err)
			}
		}
	} else {
		log.Printf("Invalid json uploaded")
		http.Error(w, "Invalid json uploaded", http.StatusBadRequest)
		return
	}

	respondWithJSON(w, http.StatusOK, s)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}