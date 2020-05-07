package utils_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"go.sirus.dev/p2p-comm/signalling/pkg/utils"
)

var _ = Describe("Token", func() {

	Context("with valid token", func() {
		It("should verify claim", func() {
			secret := "jansdandn1dandand0238r"
			claim := map[string]interface{}{
				"claim_1": "content_1",
				"claim_2": "content_2",
			}
			token, err := utils.GenerateToken(secret, claim)
			Expect(err).To(BeNil())
			Expect(token).NotTo(BeNil())
			claimResult, err := utils.ValidateToken(secret, *token)
			Expect(err).To(BeNil())
			Expect(claimResult["claim_1"]).To(Equal(claim["claim_1"]))
			Expect(claimResult["claim_2"]).To(Equal(claim["claim_2"]))
		})
	})

	Context("with invalid token", func() {
		It("should be rejected", func() {
			secret := "jansdandn1dandand0238r"
			token := "wrong_token"
			_, err := utils.ValidateToken(secret, token)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("with invalid secret", func() {
		It("should be rejected", func() {
			secret := "jansdandn1dandand0238r"
			secret2 := "asdfghkpom7829d92dad"
			claim := map[string]interface{}{
				"claim_1": "content_1",
				"claim_2": "content_2",
			}
			token, err := utils.GenerateToken(secret, claim)
			Expect(err).To(BeNil())
			Expect(token).NotTo(BeNil())
			_, err = utils.ValidateToken(secret2, *token)
			Expect(err).NotTo(BeNil())
		})
	})
})
