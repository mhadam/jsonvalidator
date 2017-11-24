package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestJsonvalidate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jsonvalidate Suite")
}
