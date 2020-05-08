package room_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRoom(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Room Management Suite")
}
