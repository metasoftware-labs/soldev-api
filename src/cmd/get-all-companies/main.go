package main

import (
	"api/internal/database"
	"api/internal/utils"
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response represents an API Gateway response
type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

// Handler AWS Lambda
func Handler(ctx context.Context, request Request) (Response, error) {

	db, err := database.GetConnection()
	defer db.Close()

	if err != nil {
		log.Println(err)
		return Response(utils.APIGateway500(errors.New("error connecting to db"))), nil
	}

	companies, err := database.GetAllCompanies(db)
	if err != nil {
		log.Println(err)
		return Response(utils.APIGateway500(errors.New("db error"))), nil
	}

	response, err := json.Marshal(&companies)
	if err != nil {
		log.Println(err)
		return Response(utils.APIGateway500(errors.New("error unmarshalling"))), nil
	}

	return Response(utils.APIGateway200Cache(response)), nil
}

func main() {
	lambda.Start(Handler)
}
