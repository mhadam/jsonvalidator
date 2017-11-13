package main

import (
	"database/sql"

	"github.com/gorilla/mux"
	"github.com/xeipuuv/gojsonschema"
	_ "github.com/lib/pq"
	"fmt"
	"log"
	"net/http"
	"encoding/json"
	"io/ioutil"
	"bufio"
	"os"
	"bytes"
	"regexp"
	"io"
)

type App struct {
	Router *mux.Router
	DB     *sql.DB
}

type AppResponse struct {
	Action string `json:"action"`
	Id string `json:"id"`
	Status string `json:"status"`
	Message string `json:"message,omitempty"`
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
	a.Router.HandleFunc("/clean", a.cleanDocumentHandler).Methods("POST")
}

func StreamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

type MapStr map[string]interface{}

func (ms *MapStr) cleanInterfaceMap(in map[interface{}]interface{}) error {
	result := make(map[string]interface{})
	result = cleanUpInterfaceMap(in)
	parseMap(result)
	*ms = result
	return nil
}

func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		if v != nil {
			result[i] = cleanUpMapValue(v)
		}
	}
	return result
}

func cleanUpInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	log.Println("cleaning interface map")
	result := make(map[string]interface{})
	for k, v := range in {
		if v == nil {
			log.Println("deleting nil map key-value")
			//delete(in, k)
		} else {
			log.Println(fmt.Sprintf("%v", k))
			result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
		}
	}
	return result
}

func cleanUpMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func parseMap(aMap map[string]interface{}) {
	for key, val := range aMap {
		switch concreteVal := val.(type) {
		case map[string]interface{}:
			log.Println(key)
			parseMap(val.(map[string]interface{}))
		case []interface{}:
			log.Println(key)
			parseArray(val.([]interface{}))
		default:
			if concreteVal == nil {
				//delete(aMap, key)
				log.Println("deleting: ", key)
			}
			log.Println(key, ":", concreteVal)
		}
	}
}

func parseArray(anArray []interface{}) {
	for i, val := range anArray {
		switch concreteVal := val.(type) {
		case map[string]interface{}:
			fmt.Println("Index:", i)
			parseMap(val.(map[string]interface{}))
		case []interface{}:
			fmt.Println("Index:", i)
			parseArray(val.([]interface{}))
		default:
			fmt.Println("Index", i, ":", concreteVal)

		}
	}
}

func cleanDocument(document []byte) ([]byte, error) {
	var raw map[interface{}]interface{}
	var cleanedUp MapStr
	json.Unmarshal(document, raw)
	cleanedUp.cleanInterfaceMap(raw)
	cleanDoc, err := json.Marshal(cleanedUp)

	return cleanDoc, err
}

func cleanDocumentRegex(document []byte) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewBuffer(document))
	cleanDoc := bytes.Buffer{}
	re := regexp.MustCompile(`:\s*null\s*,?$`)
	for scanner.Scan() {
		if re.FindStringIndex(scanner.Text()) == nil {
			cleanDoc.WriteString(scanner.Text())
		}
	}

	err := scanner.Err()
	if err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	return cleanDoc.Bytes(), err
}

func (a *App) cleanDocumentHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		respondWithString(w, http.StatusBadRequest, err.Error())
		return
	}
	defer r.Body.Close()

	log.Print(string(body))

	cleanDoc, _ := cleanDocument(body)

	respondWithString(w, http.StatusOK, string(cleanDoc))
}

func (a *App) createSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["schemaID"]
	s := jsonSchema{SchemaID: id}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		respondToInvalidSchema(w, id)
		return
	}
	defer r.Body.Close()

	if json.Valid(body) {
		s.SchemaDef = string(body)
	} else {
		log.Printf("Invalid json uploaded")
		respondToInvalidSchema(w, id)
		return
	}

	if err := s.createSchema(a.DB); err != nil {
		respondWithError(w, http.StatusBadRequest, "uploadSchema", id, err.Error())
		return
	}
	respondToValidSchema(w, id)
}

func (a *App) getSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["schemaID"]
	s := jsonSchema{SchemaID: id}

	if err := s.getSchema(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "downloadSchema", id, err.Error())
		default:
			respondWithError(w, http.StatusInternalServerError, "downloadSchema", id, err.Error())
		}
		return
	}
	respondWithString(w, http.StatusOK, s.SchemaDef)
}

func (a *App) validateSchema(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["schemaID"]
	s := jsonSchema{SchemaID: id}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		respondToInvalidDocument(w, id, err.Error())
		return
	}
	defer r.Body.Close()

	if json.Valid(body) {
		body, err = cleanDocument(body)
		if err != nil {
			log.Printf("Error cleaning document: %v", err)
			respondWithError(w, http.StatusBadRequest, "validateDocument", id, err.Error())
			return
		}
		jsonDoc := string(body)

		if err := s.getSchema(a.DB); err != nil {
			respondWithError(w, http.StatusInternalServerError, "validateDocument", id, err.Error())
			return
		}

		schemaLoader := gojsonschema.NewStringLoader(s.SchemaDef)
		documentLoader := gojsonschema.NewStringLoader(jsonDoc)
		result, err := gojsonschema.Validate(schemaLoader, documentLoader)

		if err != nil {
			respondWithError(w, http.StatusConflict, "validateDocument", id, err.Error())
			return
		}

		if !(result.Valid()) {
			for _, err := range result.Errors() {
				// Err implements the ResultError interface
				fmt.Printf("- %s\n", err)
			}
			respondWithError(w, http.StatusBadRequest, "validateDocument", id, err.Error())
		}
	} else {
		log.Printf("Invalid json document uploaded")
		respondWithError(w, http.StatusBadRequest, "validateDocument", id, "Invalid json document")
		return
	}

	respondToValidDocument(w, id)
}

func respondToInvalidSchema(w http.ResponseWriter, id string) {
	respondWithJSON(w, http.StatusBadRequest, AppResponse{
		Action: "uploadSchema",
		Id: id,
		Message: "Invalid JSON",
		Status: "error"})
}

func respondToValidSchema(w http.ResponseWriter, id string) {
	respondWithJSON(w, http.StatusCreated, AppResponse{
		Action: "uploadSchema",
		Id: id,
		Status: "success"})
}

func respondToValidDocument(w http.ResponseWriter, id string) {
	respondWithJSON(w, http.StatusOK, AppResponse{
		Action: "validateDocument",
		Id: id,
		Status: "success"})
}

func respondToInvalidDocument(w http.ResponseWriter, id string, error string) {
	respondWithJSON(w, http.StatusBadRequest, AppResponse{
		Action: "validateDocument",
		Id: id,
		Message: error,
		Status: "error"})
}

func respondWithError(w http.ResponseWriter, responseCode int, action string, id string, error string) {
	respondWithJSON(w, responseCode, AppResponse{
		Action: action,
		Id: id,
		Message: error,
		Status: "error"})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func respondWithString(w http.ResponseWriter, code int, response string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(response))
}