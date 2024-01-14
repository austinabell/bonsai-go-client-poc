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

// Headers to apply to all bonsai requests.
func applyApiKey(ctx context.Context, req *http.Request) error {
	req.Header.Set("x-api-key", os.Getenv("BONSAI_API_KEY"))
	// Note: this header is only needed to create a session, but might as well include for all
	//       to future proof, as the Rust impl does this.
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
	// Request a url to upload the data to.
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

// TODO explore this API, might be better to have it as an ambiguous reader, to avoid forcing slice alloc
func waitForSession(c context.Context, client *bonsai.ClientWithResponses, sessionUuid string) ([]byte, error) {
	for {
		// Fetch status of the session.
		statusResponse, err := client.RouteSessionStatusWithResponse(c, sessionUuid)
		if err != nil {
			return nil, err
		}
		res := statusResponse.JSON200
		if res == nil {
			return nil, fmt.Errorf("session status data not included in response, status code: %s", statusResponse.Status())
		}

		if res.Status == "RUNNING" {
			if res.State == nil {
				return nil, fmt.Errorf("state not included in response")
			}
			fmt.Printf("Current status: %s - state: %s - continue polling...\n", res.Status, *res.State)
			time.Sleep(5 * time.Second)
			continue
		} else if res.Status == "SUCCEEDED" {
			receiptURL := res.ReceiptUrl
			if receiptURL == nil {
				return nil, fmt.Errorf("receipt url not included in response")
			}

			// Download the receipt.
			receiptRes, err := http.Get(*receiptURL)
			if err != nil {
				return nil, err
			}
			defer receiptRes.Body.Close()

			receipt, err := io.ReadAll(receiptRes.Body)
			if err != nil {
				return nil, err
			}

			return receipt, nil
		} else {
			if res.ErrorMsg == nil {
				return nil, fmt.Errorf("error message not included in response")
			}
			return nil, fmt.Errorf("workflow exited: %s - err: %s", res.Status, *res.ErrorMsg)
		}
	}
}

// TODO this could re-use logic from the above, but since the responses are of very different types, this is not feasible
func waitForSnark(c context.Context, client *bonsai.ClientWithResponses, sessionUuid string) (*bonsai.SnarkReceipt, error) {
	for {
		// Fetch status of the session.
		statusResponse, err := client.RouteSnarkStatusWithResponse(c, sessionUuid)
		if err != nil {
			return nil, err
		}
		res := statusResponse.JSON200
		if res == nil {
			return nil, fmt.Errorf("session status data not included in response, status code: %s", statusResponse.Status())
		}

		if res.Status == "RUNNING" {
			fmt.Printf("Current status: %s - continue polling...\n", res.Status)
			time.Sleep(5 * time.Second)
			continue
		} else if res.Status == "SUCCEEDED" {
			return res.Output, nil
		}
	}
}

func main() {
	client, err := bonsai.NewClientWithResponses("https://api.bonsai.xyz/", bonsai.WithRequestEditorFn(applyApiKey))
	if err != nil {
		log.Fatalln(err)
	}

	// Get the version info of the server, just to verify we can connect.
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

	// Upload the input for the proving session.
	inputUuid, err := uploadInput(context.TODO(), client, data)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Input UUID:", inputUuid)

	// Create a new proving session with the image and input.
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
	starkUUID := session.JSON200.Uuid

	fmt.Println("Stark UUID:", starkUUID)
	starkReceipt, err := waitForSession(context.TODO(), client, starkUUID)
	if err != nil {
		log.Fatalln(err)
	}
	// Write the receipt to a file
	err = os.WriteFile("starkReceipt.bin", starkReceipt, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("Stark receipt written to file")

	// Stark to Snark
	snarkRes, err := client.RouteSnarkCreateWithResponse(context.TODO(), bonsai.SnarkCreate{SessionId: starkUUID})
	if err != nil {
		log.Fatalln(err)
	}
	if snarkRes.JSON200 == nil {
		log.Fatalln("Error: snark session UUID not included in response")
	}
	snarkUUID := snarkRes.JSON200.Uuid

	fmt.Println("Snark UUID:", snarkUUID)
	snarkReceipt, err := waitForSnark(context.TODO(), client, snarkUUID)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("Snark receipt: %+v\n", snarkReceipt)
}
