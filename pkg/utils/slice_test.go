package utils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"syreclabs.com/go/faker"

	"go.sirus.dev/p2p-comm/signalling/pkg/utils"
)

var _ = Describe("Slice", func() {
	Describe("ContainString", func() {
		When("string in slice", func() {
			It("should return true", func() {
				present := faker.Lorem().Characters(14)
				ok := utils.ContainString([]string{
					present,
					faker.Lorem().Characters(12),
					faker.Lorem().Characters(6),
					faker.Lorem().Characters(21),
				}, present)
				Expect(ok).To(BeTrue())
			})
		})
		When("string not in slice", func() {
			It("should return false", func() {
				ok := utils.ContainString([]string{
					faker.Lorem().Characters(14),
					faker.Lorem().Characters(12),
					faker.Lorem().Characters(6),
					faker.Lorem().Characters(21),
				}, "non-exist-text")
				Expect(ok).To(BeFalse())
			})
		})
	})
})
