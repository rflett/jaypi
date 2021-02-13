package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/skip2/go-qrcode"
)

const inviteLink = "https://www.fuckYouNorto.com/invite?groupCode=%s"

// RequestDto is the expected body in the request
type RequestDto struct {
	UserID string `json:"userId"`
}

// ResponseDto is the response that we'll send back
type ResponseDto struct {
	Link   string `json:"link"`
	QRCode string `json:"qrCode"`
}

// Handler is our handle on life
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// get inviteCode from request
	requestDto := RequestDto{}
	err := json.Unmarshal([]byte(request.Body), &requestDto)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
	}

	// make link
	link := fmt.Sprintf(inviteLink, requestDto.UserID)

	// make qr groupCode
	qrCode, err := qrcode.Encode(link, qrcode.Low, 256)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 400}, nil
	}

	// create and send the response
	responseDto := ResponseDto{
		Link:   link,
		QRCode: base64.StdEncoding.EncodeToString(qrCode),
	}
	body, _ := json.Marshal(responseDto)
	headers := map[string]string{"Content-Type": "application/json"}
	return events.APIGatewayProxyResponse{Body: string(body), StatusCode: 200, Headers: headers}, nil
}

func main() {
	lambda.Start(Handler)
}
