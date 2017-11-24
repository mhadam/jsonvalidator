package main_test

import (
	. "github.com/jsonvalidate"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"io/ioutil"
	"fmt"
	"bytes"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http/httptest"
)

var _ = Describe("App", func() {
	var a App
	var router *mux.Router
	var recorder *httptest.ResponseRecorder
	var request *http.Request

	BeforeEach(func() {
		a = App{}
		a.Run(":8080")
		a.Initialize(
			"postgres",
			"snowplow",
			"snowplow",
		)

		recorder = httptest.NewRecorder()
		router = a.Router
	})

	AfterEach(func() {
		a.ensureTableExists()
		a.clearTable()
	})

	Describe("POST /create-schema", func() {
		raw, err := ioutil.ReadFile("./config-schema-valid.json")
		if err != nil {
			fmt.Println(err.Error())
		}

		BeforeEach(func() {
			request, _ = http.NewRequest("POST", "/create-schema", bytes.NewReader(raw))
		})

		It("returns a status code of 201", func() {
			router.ServeHTTP(recorder, request)
			Expect(recorder.Code).To(Equal(http.StatusCreated))
		})

		var m map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &m)

		It("returns correct body", func() {
			Expect(m["action"]).To(Equal("uploadSchema"))
			Expect(m["id"]).To(Equal("config-schema"))
			Expect(m["status"]).To(Equal("success"))
		})
	})
})
