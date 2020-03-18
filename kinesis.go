package main

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/kinesis/kinesisiface"
	uuid "github.com/nu7hatch/gouuid"
)

const (
	defaultRegion = "us-west-2"
	chunkSize     = 1048576 - 200
)

func getRegion() string {
	var region string
	if os.Getenv("AWS_REGION") != "" {
		region = os.Getenv("AWS_REGION")
	} else {
		region = defaultRegion
	}
	return region
}

func buildKinesisClient(streamARN string) *kinesis.Kinesis {
	userSession := session.Must(session.NewSession(&aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(verboseCredentialErrors),
		Region:                        aws.String(getRegion()),
	}))
	var kclient *kinesis.Kinesis
	log.Printf("Creating Kinesis client")
	// Allows for dev override
	endpoint := os.Getenv("STREAM_ENDPOINT")
	if endpoint != "" {
		log.Printf("Sending unverified traffic to stream endpoint=" + endpoint)
		kclient = kinesis.New(userSession, &aws.Config{
			Endpoint:    &endpoint,
			Credentials: credentials.NewStaticCredentials("x", "x", "x"),
			Region:      aws.String(getRegion()),
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true},
				}},
		})
	} else {
		log.Println("Fetching temp credentials...")
		kinesisTempCreds := stscreds.NewCredentials(userSession, streamARN)
		log.Println("Success!")
		kclient = kinesis.New(userSession, &aws.Config{
			CredentialsChainVerboseErrors: aws.Bool(verboseCredentialErrors),
			Credentials:                   kinesisTempCreds,
			Region:                        aws.String(getRegion()),
		})
	}
	return kclient
}

func chunkData(data string, size int) []string {
	chunks := []string{}
	for c := 0; c < len(data); c += size {
		nextChunk := data[c:min(c+size, len(data))]
		chunks = append(chunks, nextChunk)
	}
	return chunks
}

func min(i1, i2 int) int {
	if i1 < i2 {
		return i1
	}
	return i2
}

func buildMessages(line string) []EventChunk {
	chunks := chunkData(line, chunkSize)
	numChunks := len(chunks)
	messages := []EventChunk{}
	eventUUID, err := uuid.NewV4()
	var correlation string
	if err != nil {
		msg := fmt.Sprintf("UUID generation failed: %s\nFalling back to SHA1 of input string for chunk correlation", err)
		logDebug(msg)
		correlation = fmt.Sprintf("%x", sha1.Sum([]byte(line)))
	} else {
		correlation = eventUUID.String()
	}
	for chunkID, chunk := range chunks {
		nextMessage := EventChunk{
			ChunkNumber: chunkID,
			NumChunks:   numChunks,
			UUID:        correlation,
			Data:        chunk,
		}
		messages = append(messages, nextMessage)
	}
	return messages
}

func sendToStream(message interface{}, stream string, client kinesisiface.KinesisAPI) error {
	dataBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}
	partition := "replay-partition-key-" + time.Now().String()
	response, err := client.PutRecord(&kinesis.PutRecordInput{
		StreamName:   aws.String(stream),
		Data:         dataBytes,
		PartitionKey: &partition,
	})
	if err != nil {
		return err
	}
	log.Printf("%+v\n", response)
	return nil
}
