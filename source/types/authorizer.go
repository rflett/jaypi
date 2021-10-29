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
	const RyanUserID = "2ef05ca2-aef4-40aa-8e5f-d69c7795e543"
	const JamesUserID = "0f74f03e-0a04-463e-a577-4cce146ff670"

	admins := []string{RyanUserID, JamesUserID}

	for _, admin := range admins {
		if a.UserID == admin {
			return nil
		}
	}
	return errors.New("User is not an administrator.")
}
