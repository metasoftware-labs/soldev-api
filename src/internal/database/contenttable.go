package database

import (
	"api/internal/types"
	"api/internal/utils"
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// SaveContent saves Content in the DB
func SaveContent(content types.Content) error {
	// TODO: Handle error
	dynamo, _ := Client()

	// content.SpecialTag is a GSI but sometimes the UI
	// sends an empty string in the field. This is
	// a quick fix for someone who is being lazy (myself)
	if len(content.SpecialTag) == 0 {
		content.SpecialTag = "0"
	}

	// We send data to Algolia only when is "active", and delete data from Algolia
	// when "inactive".
	algoliaContent := types.AlgoliaRecord{ObjectID: content.SK, Content: content}
	if content.ContentStatus == "active" {
		// Save Data in Algolia
		_, err := AlgoliaIndex().SaveObjects(algoliaContent)
		if err != nil {
			log.Printf("Error while saving data in Algolia: %v ", err)
			return err
		}
	} else if content.ContentStatus == "inactive" {
		// Delete data from Algolia
		_, err := AlgoliaIndex().DeleteObject(content.SK)
		if err != nil {
			log.Printf("Error while deleting data in Algolia: %v ", err)
			return err
		}
	}

	data, err := attributevalue.MarshalMap(content)
	if err != nil {
		log.Printf("Error while marshalling: %v ", err)
		return err
	}

	_, err = dynamo.PutItem(context.TODO(), &dynamodb.PutItemInput{
		Item:      data,
		TableName: aws.String("Content"),
	})

	if err != nil {
		log.Printf("Error while saving data: %v ", err)
		return err
	}

	return nil
}

// DeleteContent ...
func DeleteContent(content types.Content) error {
	// TODO: Handle error
	dynamo, _ := Client()

	key, err := attributevalue.MarshalMap(struct {
		PK string
		SK string
	}{
		content.PK,
		content.SK,
	})

	if err != nil {
		log.Printf("Error while marshaling composite key: %v", err)
		return err
	}

	_, err = dynamo.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String("Content"),
		Key:       key,
	})

	if err != nil {
		log.Printf("Error while deleting data: %v ", err)
		return err
	}

	return nil
}

// QueryContent ...
func QueryContent(vertical string, contentType string, specialTag string, list string) ([]types.Content, bool, error) {
	// TODO: Handle error
	dynamo, _ := Client()
	var contentList []types.Content

	var result *dynamodb.QueryOutput
	var err error

	// Find out if the content type is video/playlist, in that case we need to use
	// a specific gsi to sort by publishedAt timestamp
	videoContent := true
	for _, item := range utils.GetContentTypes() {
		if item == contentType {
			// content type is not video/playlist
			videoContent = false
			break
		}
	}

	if len(contentType) > 0 && videoContent {
		result, err = dynamo.Query(context.TODO(), &dynamodb.QueryInput{
			TableName:              aws.String("Content"),
			IndexName:              aws.String("video-gsi"),
			KeyConditionExpression: aws.String("PK = :PK"),
			ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
				":PK": &dynamoTypes.AttributeValueMemberS{Value: vertical + "#" + contentType},
			},
		})
	} else if len(specialTag) != 0 {
		// If specialTag is provided as a parameter we need to use the correct GSI
		result, err = dynamo.Query(context.TODO(), &dynamodb.QueryInput{
			TableName:              aws.String("Content"),
			KeyConditionExpression: aws.String("SpecialTag = :SpecialTag"),
			IndexName:              aws.String("special-tag-gsi"),
			ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
				":SpecialTag": &dynamoTypes.AttributeValueMemberS{Value: specialTag},
			},
			ScanIndexForward: aws.Bool(false),
		})
	} else {
		result, err = dynamo.Query(context.TODO(), &dynamodb.QueryInput{
			TableName:              aws.String("Content"),
			KeyConditionExpression: aws.String("PK = :PK"),
			ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
				":PK": &dynamoTypes.AttributeValueMemberS{Value: vertical + "#" + contentType},
			},
		})
	}

	if err != nil {
		log.Printf("Error while running query to get content: %v", err)
		return nil, videoContent, err
	}

	if len(result.Items) == 0 {
		return nil, videoContent, errors.New("404")
	}

	for _, i := range result.Items {
		var content types.Content
		err := attributevalue.UnmarshalMap(i, &content)
		if err != nil {
			log.Printf("Error unmarshalling: %v ", err)
			return nil, videoContent, err
		}

		// If content status != active we need to ignore it
		if content.ContentStatus != "active" {
			continue
		}

		contentList = append(contentList, content)
	}

	return contentList, videoContent, nil
}

// ScanContent ...
func ScanContent() ([]types.Content, error) {
	// TODO: Handle error
	dynamo, _ := Client()
	var contentList []types.Content

	result, err := dynamo.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName:        aws.String("Content"),
		FilterExpression: aws.String("ContentType <> :ContentType"),
		ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
			":ContentType": &dynamoTypes.AttributeValueMemberS{Value: "Playlist"},
		},
	})

	if err != nil {
		log.Printf("Error while running query to get content: %v", err)
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, errors.New("404")
	}

	for _, i := range result.Items {
		var content types.Content
		err := attributevalue.UnmarshalMap(i, &content)
		if err != nil {
			log.Printf("Error unmarshalling: %v ", err)
			return nil, err
		}

		// If content status != active we need to ignore it
		if content.ContentStatus != "active" {
			continue
		}

		contentList = append(contentList, content)
	}

	return contentList, nil

}

// GetContentDetails ...
func GetContentDetails(pk string, sk string) (types.Content, error) {
	// TODO: Handle error
	dynamo, _ := Client()
	var content types.Content

	key, err := attributevalue.MarshalMap(struct {
		PK string
		SK string
	}{
		pk,
		sk,
	})

	if err != nil {
		log.Printf("Error while marshaling composite key: %v", err)
		return types.Content{}, err
	}

	result, err := dynamo.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String("Content"),
		Key:       key,
	})

	if err != nil {
		log.Printf("Error while running query to get content using playlistID+ContentID: %v", err)
		return types.Content{}, err
	}

	// If we did not found any content it's because
	// the playlist OR the content does not exist
	if result.Item == nil {
		return types.Content{}, errors.New("404")
	}

	err = attributevalue.UnmarshalMap(result.Item, &content)
	if err != nil {
		log.Printf("Error unmarshalling: %v ", err)
		return types.Content{}, err
	}

	return content, nil
}

// QueryContentByStatus ...
func QueryContentByStatus(status string) ([]types.Content, error) {
	// TODO: Handle error
	dynamo, _ := Client()
	var contentList []types.Content

	result, err := dynamo.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String("Content"),
		IndexName:              aws.String("status-gsi"),
		KeyConditionExpression: aws.String("ContentStatus = :status"),
		ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
			":status": &dynamoTypes.AttributeValueMemberS{Value: status},
		},
	})

	if err != nil {
		log.Printf("Error while requesting live content: %v", err)
		return nil, err
	}

	for _, i := range result.Items {
		var content types.Content
		err := attributevalue.UnmarshalMap(i, &content)
		if err != nil {
			log.Printf("Error unmarshalling: %v ", err)
			return nil, err
		}
		contentList = append(contentList, content)
	}

	return contentList, nil
}

// GetContentByUrl finds all content associated with a specific URL
func GetContentByUrl(url string) ([]types.Content, error) {
	// TODO: Handle error
	dynamo, _ := Client()
	var contentList []types.Content

	result, err := dynamo.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String("Content"),
		IndexName:              aws.String("url-gsi"),
		KeyConditionExpression: aws.String("#ContentUrl = :ContentUrl"),
		ExpressionAttributeNames: map[string]string{
			"#ContentUrl": "Url",
		},
		ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
			":ContentUrl": &dynamoTypes.AttributeValueMemberS{Value: url},
		},
	})

	if err != nil {
		log.Printf("Error while requesting live content: %v", err)
		return nil, err
	}

	for _, i := range result.Items {
		var content types.Content
		err := attributevalue.UnmarshalMap(i, &content)
		if err != nil {
			log.Printf("Error unmarshalling: %v ", err)
			return nil, err
		}
		contentList = append(contentList, content)
	}

	return contentList, nil
}
