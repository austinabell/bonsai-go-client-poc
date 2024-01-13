package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/austinabell/bonsai"
)

func applyApiKey(ctx context.Context, req *http.Request) error {
	req.Header.Set("x-api-key", "")
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
