package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/brandonyeoxg/einocodex"
	"github.com/cloudwego/eino/schema"
	"github.com/openai/openai-go/v3"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	m, err := einocodex.NewModel(ctx, &einocodex.ResponsesConfig{
		Model: "gpt-5.6-terra",
	})
	if err != nil {
		log.Fatal(err)
	}

	stream, err := m.Stream(ctx, []*schema.AgenticMessage{
		schema.UserAgenticMessage("Reply only hello world."),
	})
	if err != nil {
		var apiErr *openai.Error
		if errors.As(err, &apiErr) && apiErr.Response != nil {
			body, _ := io.ReadAll(io.LimitReader(apiErr.Response.Body, 4096))
			log.Printf("400 response body: %s", body)
		}
		log.Fatal(err)
	}
	defer stream.Close()

	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			log.Fatal(err)
			break
		}
		for _, block := range msg.ContentBlocks {
			if block.AssistantGenText != nil {
				fmt.Print(block.AssistantGenText.Text)
			}
		}
	}
	fmt.Println()
}
