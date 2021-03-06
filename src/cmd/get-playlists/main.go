package main

import (
	"api/internal/modules"
	"api/internal/utils"
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response represents an API Gateway response
type Response events.APIGatewayProxyResponse

// Request represents an API Gateway request
type Request events.APIGatewayProxyRequest

// Handler AWS Lambda
func Handler(ctx context.Context, request Request) (Response, error) {
	var vertical = request.PathParameters["vertical"]

	// Get all saved playlists
	playlists, err := modules.GetPlaylists(vertical)
	// Respond 500 if getting channels from database failed
	if err != nil {
		return Response(utils.APIGateway500(err)), nil
	}

	if playlists == nil {
		return Response(utils.APIGateway200([]byte{})), nil
	}

	// Marshal the response into json bytes
	response, err := json.Marshal(&playlists)
	if err != nil {
		return Response(utils.APIGateway500(err)), nil
	}

	return Response(utils.APIGateway200(response)), nil
}

func main() {
	lambda.Start(Handler)
}
