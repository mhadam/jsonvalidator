package main

import (
	"os"
	"testing"

	"log"
	"net/http"
	"net/http/httptest"
	"encoding/json"
	"io/ioutil"
	"fmt"
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
	a.DB.Exec("TRUNCATE json_schema")
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

func TestCreateSchema(t *testing.T) {
	clearTable()

	raw, err := ioutil.ReadFile("./pages.json")
	if err != nil {
		fmt.Println(err.Error())
	}

	req, _ := http.NewRequest("POST", "/schema/config-schema", bytes.NewBuffer(raw))
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