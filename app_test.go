package main_test

import (
	. "github.com/jsonvalidate"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/Benjamintf1/Expanded-Unmarshalled-Matchers"
	"net/http"
	"io/ioutil"
	"fmt"
	"bytes"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"encoding/json"
	"log"
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

	Describe("POST /clean", func() {
		var (
			testFile []byte
			cleanTestFile []byte
			err error
		)

		BeforeEach(func() {
			testFile, err = ioutil.ReadFile("./config.json")
			if err != nil {
				fmt.Println(err.Error())
			}

			cleanTestFile, err = ioutil.ReadFile("./clean-config.json")
			if err != nil {
				fmt.Println(err.Error())
			}

			request, _ = http.NewRequest("POST", "/clean", bytes.NewReader(testFile))
			router.ServeHTTP(recorder, request)
		})

		It("returns a status code of 200", func() {
			Expect(recorder.Code).To(Equal(http.StatusOK))
		})

		It("returns correct json", func() {
			Expect(recorder.Body.Bytes()).Should(MatchUnorderedJSON(cleanTestFile))
		})
	})

	Describe("POST /validate/config-schema", func() {
		var (
			validSchemaFile []byte
			configFile []byte
			err error
			m map[string]interface{}
		)

		BeforeEach(func() {
			validSchemaFile, err = ioutil.ReadFile("./config-schema-valid.json")
			if err != nil {
				fmt.Println(err.Error())
			}

			configFile, err = ioutil.ReadFile("./config.json")
			if err != nil {
				fmt.Println(err.Error())
			}

			request, _ = http.NewRequest("POST", "/schema/config-schema", bytes.NewReader(configFile))
			router.ServeHTTP(recorder, request)

			recorder = httptest.NewRecorder()
			request, _ = http.NewRequest("POST", "/validate/config-schema", bytes.NewBuffer(validSchemaFile))
			router.ServeHTTP(recorder, request)
			log.Println(string(recorder.Body.Bytes()))
		})

		It("returns a status code of 200", func() {
			Expect(recorder.Code).To(Equal(http.StatusOK))
		})

		It("returns correct body", func() {
			json.Unmarshal(recorder.Body.Bytes(), &m)
			Expect(m["action"]).To(Equal("validateDocument"))
			Expect(m["id"]).To(Equal("config-schema"))
			Expect(m["status"]).To(Equal("success"))
		})
	})
})
