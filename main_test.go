package main

import (
	"os"
	"testing"

	"log"
	"net/http"
	"net/http/httptest"
	"encoding/json"
	"bytes"
)

var a App

func TestMain(m *testing.M) {
	a = App{}
	a.Initialize(
		"postgres",
		"snowplow",
		"snowplow")

	ensureTableExists()

	code := m.Run()

	clearTable()

	os.Exit(code)
}

func ensureTableExists() {
	if _, err := a.DB.Exec(tableCreationQuery); err != nil {
		log.Fatal(err)
	}
}

func clearTable() {
	a.DB.Exec("TRUNCATE schemas")
	//shouldn't be necessary with restart identity: a.DB.Exec("ALTER SEQUENCE schemas_id_seq RESTART WITH 1")
}

const tableCreationQuery = `CREATE TABLE IF NOT EXISTS json_schema
(
json_schema_id TEXT PRIMARY KEY,
json_schema_def TEXT
)`

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestEmptyTable(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/schema", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); body != "[]" {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}

func TestGetNonExistentSchema(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("GET", "/schema/11", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	json.Unmarshal(response.Body.Bytes(), &m)
	if m["error"] != "Schema not found" {
		t.Errorf("Expected the 'error' key of the response to be set to 'Schema not found'. Got '%s'", m["error"])
	}
}

func TestCreateSchema(t *testing.T) {
	clearTable()

	req, _ := http.NewRequest("POST", "/schema", bytes.NewBuffer(payload))
	response := executeRequest(req)

	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	if m["action"] != "uploadSchema" {
		t.Errorf("Expected action to be 'uploadSchema'. Got '%v'", m["action"])
	}

	if m["id"] != "config-schema" {
		t.Errorf("Expected id to be 'config-schema'. Got '%v'", m["id"])
	}

	if m["status"] != "success" {
		t.Errorf("Expected status to be 'success'. Got '%v'", m["status"])
	}
}