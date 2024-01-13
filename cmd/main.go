package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/austinabell/bonsai"
)

func applyApiKey(ctx context.Context, req *http.Request) error {
	apiKey := os.Getenv("BONSAI_API_KEY")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("x-risc0-version", "0.19.1")
	return nil
}

func putDataToURL(c context.Context, url string, data io.Reader) error {
	// TODO find cleaner way to handle this. Ideally reusing same client.
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, url, data)
	if err != nil {
		return err
	}
	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func uploadInput(c context.Context, client *bonsai.ClientWithResponses, data []byte) (string, error) {
	uploadResponse, err := client.RouteInputUploadWithResponse(c)
	if err != nil {
		return "", err
	}
	uploadData := uploadResponse.JSON200
	if uploadData == nil {
		return "", fmt.Errorf("upload data not included in response")
	}

	// Upload the data to the url provided by the server.
	err = putDataToURL(c, uploadData.Url, bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}

	return uploadData.Uuid, nil
}

func main() {
	client, err := bonsai.NewClientWithResponses("https://api.bonsai.xyz/", bonsai.WithRequestEditorFn(applyApiKey))
	if err != nil {
		log.Fatalln(err)
	}
	response, err := client.RouteVersionDataWithResponse(context.TODO())
	if err != nil {
		log.Fatalln(err)
	}
	// fmt.Println("Status:", response.Status(), response.StatusCode())
	fmt.Println("Version Info:", response.JSON200)

	var (
		inputA uint64 = 7
		inputB uint64 = 3
	)

	// TODO this is just handling what risc0 serialization would do. Likely better to use a
	// 		different serialization protocol that has a golang equivalent.
	data := make([]byte, 16)
	binary.LittleEndian.PutUint64(data[:8], inputA)
	binary.LittleEndian.PutUint64(data[8:], inputB)

	inputUuid, err := uploadInput(context.TODO(), client, data)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Input UUID:", inputUuid)

	sessionCreateParams := bonsai.SessionCreate{
		// Note: Image ID is coming from examples/factors.
		Img:   "d5ddf6ddb4b8ee860a56c4fcf65d0b1b843ac2aa74b45c0d0b71d8c1db424ecb",
		Input: inputUuid,
	}
	session, err := client.RouteSessionCreateWithResponse(context.TODO(), sessionCreateParams)
	if err != nil {
		log.Fatalln(err)
	}

	// fmt.Println("Session:", session.Status(), session.StatusCode())
	if session.JSON200 == nil {
		log.Fatalln("Error: created session data not included in response")
	}
	sessionUuid := session.JSON200.Uuid

	for {
		statusResponse, err := client.RouteSessionStatusWithResponse(context.TODO(), sessionUuid)
		if err != nil {
			log.Fatalln(err)
		}
		res := statusResponse.JSON200
		if res == nil {
			log.Fatalln("Error: session status data not included in response")
		}

		if res.Status == "RUNNING" {
			if res.State == nil {
				log.Fatalln("Error: state not included in response")
			}
			fmt.Printf("Current status: %s - state: %s - continue polling...\n", res.Status, *res.State)
			time.Sleep(5 * time.Second)
			continue
		}

		if res.Status == "SUCCEEDED" {
			receiptURL := res.ReceiptUrl
			if receiptURL == nil {
				log.Fatalln("Error: receipt url not included in response")
			}

			// Download the receipt.
			receiptRes, err := http.Get(*receiptURL)
			if err != nil {
				log.Fatalln(err)
			}
			defer receiptRes.Body.Close()

			receipt, err := io.ReadAll(receiptRes.Body)
			if err != nil {
				log.Fatalln(err)
			}

			// Write the receipt to a file
			err = os.WriteFile("receipt.bin", receipt, 0644)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println("Receipt written to file")

			// var receipt Receipt
			// err = bincode.Deserialize(receiptBuf, &receipt)
			// if err != nil {
			// 	log.Fatalln(err)
			// }

			// err = receipt.Verify(METHOD_NAME_ID)
			// if err != nil {
			// 	log.Fatalln(err)
			// }
		} else {
			if res.ErrorMsg == nil {
				log.Fatalln("Error: error message not included in response")
			}
			log.Fatalf("Workflow exited: %s - err: %s\n", res.Status, *res.ErrorMsg)
			return
		}

		break
	}
}
