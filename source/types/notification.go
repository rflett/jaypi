package types

import (
	"encoding/base64"
	"encoding/json"
)

var APNSKey = "APNS"

type Notification struct {
	Title      string `json:"title"`
	Message    string `json:"message"`
	CustomData string `json:"customData"` // TODO this would be our notification data
}

// TODO this looks horrendous
func (n *Notification) AndroidPayload() string {
	var gcmPayload = make(map[string]interface{})
	gcmPayload["data"] = map[string]string{
		"message":    n.Message,
		"customData": n.CustomData,
	}

	marshalledGcm, _ := json.Marshal(gcmPayload)
	messagePayload := map[string]string{
		"GCM": string(marshalledGcm),
	}

	marshalledMessage, _ := json.Marshal(messagePayload)
	return string(marshalledMessage)
}

// TODO this looks horrendous
func (n *Notification) IosPayload() string {
	var apsPayload = make(map[string]interface{})
	apsPayload["alert"] = map[string]string{
		"title": n.Title,
		"body":  n.Message,
	}

	data := map[string]string{
		"customData": n.CustomData,
	}
	marshalledData, _ := json.Marshal(data)

	apnsPayload := map[string]interface{}{
		"aps":  apsPayload,
		"data": base64.StdEncoding.EncodeToString(marshalledData),
	}
	marshalledApns, _ := json.Marshal(apnsPayload)
	messagePayload := map[string]interface{}{
		APNSKey: string(marshalledApns),
	}

	marshalledMessage, _ := json.Marshal(messagePayload)
	return string(marshalledMessage)
}
