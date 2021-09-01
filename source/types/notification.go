package types

import (
	"encoding/json"
)

type Notification struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	CustomData string `json:"customData"` // TODO this would be our notification data
}

func (n *Notification) AndroidPayload() string {
	// Slightly less horrendous?
	var messagePayload = map[string]map[string]map[string]interface{}{
		"GCM": {
			"data": {
				"message":    n.Message,
				"customData": n.CustomData,
			},
		},
	}
	marshalledMessage, _ := json.Marshal(messagePayload)
	return string(marshalledMessage)
}

func (n *Notification) IosPayload() string {
	// Slightly less horrendous? (Still pretty bad)
	messagePayload := map[string]map[string]interface{}{
		"APNS": {
			"aps": map[string]map[string]interface{}{
				"alert": {
					"title": n.Title,
					"body":  n.Message,
				},
			},
			"data": n.CustomData,
		},
	}

	marshalledMessage, _ := json.Marshal(messagePayload)
	return string(marshalledMessage)
}
