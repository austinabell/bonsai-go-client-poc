package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/austinabell/bonsai"
)

func applyApiKey(ctx context.Context, req *http.Request) error {
	apiKey := os.Getenv("BONSAI_API_KEY")
	req.Header.Set("x-api-key", apiKey)
	return nil
}

func main() {
	c, err := bonsai.NewClientWithResponses("https://api.bonsai.xyz/", bonsai.WithRequestEditorFn(applyApiKey))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	response, err := c.RouteVersionDataWithResponse(context.TODO())
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Status:", response.Status(), response.StatusCode())
	fmt.Println("Version Info:", response.JSON200)
}
