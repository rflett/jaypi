package types

import "errors"

type AuthorizerContext struct {
	AuthProvider   string
	AuthProviderId string
	Name           string
	UserID         string
}

func (a *AuthorizerContext) IsAdmin() error {
	// bad, but sufficient
	const RyanUserID = "2e26e7dc-3f8c-456d-9d1b-8ce5b6447585"
	const JamesUserID = "0f74f03e-0a04-463e-a577-4cce146ff670"

	admins := []string{RyanUserID, JamesUserID}

	for _, admin := range admins {
		if a.UserID == admin {
			return nil
		}
	}
	return errors.New("User is not an administrator.")
}
