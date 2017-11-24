package main_test

import (
	. "github.com/jsonvalidate"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"io/ioutil"
	"fmt"
	"bytes"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"encoding/json"
)

var _ = Describe("App", func() {
	var (
		a App
		router *mux.Router
		recorder *httptest.ResponseRecorder
		request *http.Request
	)

	BeforeEach(func() {
		a = App{}
		a.Initialize(
			"postgres",
			"snowplow",
			"snowplow",
		)

		a.EnsureTableExists()
		a.ClearTable()
		recorder = httptest.NewRecorder()
		router = a.Router
	})

	AfterEach(func() {
		a.EnsureTableExists()
		a.ClearTable()
	})

	Describe("POST /create-schema", func() {
		var (
			m map[string]interface{}
		)

		BeforeEach(func() {
			raw, err := ioutil.ReadFile("./config-schema-valid.json")
			if err != nil {
				fmt.Println(err.Error())
			}

			request, _ = http.NewRequest("POST", "/schema/config-schema", bytes.NewReader(raw))
			router.ServeHTTP(recorder, request)
		})

		It("returns a status code of 201", func() {
			Expect(recorder.Code).To(Equal(http.StatusCreated))
		})

		It("returns correct body", func() {
			json.Unmarshal(recorder.Body.Bytes(), &m)
			Expect(m["action"]).To(Equal("uploadSchema"))
			Expect(m["id"]).To(Equal("config-schema"))
			Expect(m["status"]).To(Equal("success"))
		})
	})
})
