package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	queueURL *string
	queue    *sqs.SQS
	authKey  string
)

const (
	headerSignature = "X-Hub-Signature"
)

func init() {
	if url := strings.TrimSpace(os.Getenv("QUEUE_URL")); url != "" {
		queueURL = &url
	}

	region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if queueURL == nil || region == "" {
		log.Fatal("QUEUE_URL and AWS_REGION must be provided")
	}
	queue = sqs.New(session.Must(session.NewSession(aws.NewConfig().WithRegion(region))))
}

type ghPayload struct {
	Payload string
	Headers map[string]string
}

var authError = fmt.Errorf("authentication failed")

func awsLambdaHandler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{}
	sign := req.Headers[headerSignature]
	h := hmac.New(sha1.New, []byte(authKey))
	h.Write([]byte(req.Body))
	sum := "sha1=" + string(hex.EncodeToString(h.Sum(nil)))
	if sum != sign {
		resp.StatusCode = http.StatusForbidden
		return resp, authError
	}

	payload := ghPayload{
		Headers: make(map[string]string),
		Payload: req.Body,
	}
	for k, v := range req.Headers {
		if !strings.HasPrefix(k, "CloudFront-") {
			payload.Headers[k] = v
		}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		resp.StatusCode = http.StatusInternalServerError
		return resp, err
	}

	out, err := queue.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    queueURL,
		MessageBody: aws.String(string(b)),
	})
	if err == nil {
		resp.StatusCode = http.StatusOK
		resp.Body = *out.MessageId
		return resp, nil
	}

	resp.StatusCode = http.StatusInternalServerError
	return resp, fmt.Errorf("failed to send to SQS: %v", err)
}

func deleteMessage(msg *sqs.Message) error {
	_, err := queue.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      queueURL,
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("Failed to delete message %s: %v", *msg.MessageId, err)
	}
	return err
}

func toJenkins(url string) error {
	for {
		out, err := queue.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            queueURL,
			VisibilityTimeout:   aws.Int64(1),
			WaitTimeSeconds:     aws.Int64(20),
			MaxNumberOfMessages: aws.Int64(3),
		})
		if err != nil {
			return err
		}
		for _, msg := range out.Messages {
			pl := ghPayload{}
			if err := json.Unmarshal([]byte(*msg.Body), &pl); err != nil {
				log.Printf("Failed to unmarshal SQS message: %v", err)
				deleteMessage(msg)
				continue
			}

			log.Printf("Got webhook message %s, event: %s", *msg.MessageId, pl.Headers["X-GitHub-Event"])

			req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(pl.Payload))
			for k, v := range pl.Headers {
				req.Header.Set(k, v)
			}
			ctxTimeout, _ := context.WithTimeout(context.Background(), time.Second*5)
			req = req.WithContext(ctxTimeout)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("Failed to relay message to jenkins: %v", err)
			} else {
				if resp.StatusCode == http.StatusOK {
					log.Printf("Successfully relayed %s event %s to jenkins", req.Header["X-Github-Event"], *msg.MessageId)
					io.Copy(ioutil.Discard, resp.Body)
				} else {
					b, _ := ioutil.ReadAll(resp.Body)
					log.Printf("Error occured while realying %s event %s to jenkins. Jenkins response: %s",
						req.Header["X-Github-Event"], *msg.MessageId, string(b))
				}
				resp.Body.Close()
			}

			deleteMessage(msg)
		}

		time.Sleep(time.Second * 5)
	}

	return nil
}

func main() {
	var mode string
	flag.StringVar(&mode, "mode", "server", "run mode. server or client")
	flag.Parse()

	if mode == "client" {
		jenkinsURL := strings.TrimSpace(os.Getenv("JENKINS_GITHUB_HOOK_URL"))
		if _, err := url.Parse(jenkinsURL); err != nil {
			log.Fatal("JENKINS_GITHUB_HOOK_URL must be provided")
		}
		log.Fatal(toJenkins(jenkinsURL))
	}

	authKey = strings.TrimSpace(os.Getenv("GITHUB_WEBHOOK_SECRET"))
	if authKey == "" || authKey == "__GITHUB_WEBHOOK_SECRET__" {
		log.Fatal("GITHUB_WEBHOOK_SECRET environment variable must be provided")
	}
	lambda.Start(awsLambdaHandler)
}
