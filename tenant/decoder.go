package tenant

import (
	"encoding/base64"
	"encoding/json"
	"log"
)

// headerContents is the main structure for the "x-rh-identity" header. {"identity": {"account_number"}}
type headerContents struct {
	Identity identity `json:"identity"`
}

// identity is the inner structure for the "x-rh-identity" header. {"identity": {"account_number"}}
type identity struct {
	AccountNumber string `json:"account_number"`
}

// DecodeAccountNumber decodes the account number from the "x-rh-identity" header contents.
func DecodeAccountNumber(xRhIdentity string) string {
	contents, err := base64.StdEncoding.DecodeString(xRhIdentity)
	if err != nil {
		log.Printf(`could not decode base64 contents from "%s"`, xRhIdentity)
		return ""
	}

	var accountNumber headerContents
	err = json.Unmarshal(contents, &accountNumber)
	if err != nil {
		log.Printf(`could not extract account number from "x-rh-identity" header: %s`, err)
		return ""
	}

	return accountNumber.Identity.AccountNumber
}
