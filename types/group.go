package types

import (
	"encoding/json"
	"fmt"
)

// Group is a collection of users that can view each others song guesses
type Group struct {
	ID       string
	Owner    int
	Nickname string
	Members  []int
}

// add creates the Group in the database
func (g Group) add() (string, error) {
	groupJSON, err := json.Marshal(g)
	fmt.Println(string(groupJSON))
	return g.ID, err
}
