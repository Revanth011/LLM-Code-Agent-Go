package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"encoding/json"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// type ToolCall struct {
// 	Id string `json:"id"`
// 	Type string `json:"type"`
// 	Function struct {
// 		Name string `json:"name"`
// 		Arguments string `json:"arguments"`
// 	} `json:"function"`
// }

func Read(file_path string) string {
	data := make([]byte, 100)
	file, err := os.Open(file_path)
	if err != nil {
		log.Fatal(err)
	}

	_, err = file.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
}

func main() {
	var prompt string
	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.Parse()

	if prompt == "" {
		panic("Prompt must not be empty")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")
	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}

	if apiKey == "" {
		panic("Env variable OPENROUTER_API_KEY not found")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseUrl))
	resp, err := client.Chat.Completions.New(context.Background(),
		openai.ChatCompletionNewParams{
			Model: "anthropic/claude-haiku-4.5",
			Messages: []openai.ChatCompletionMessageParamUnion{
				{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(prompt),
						},
					},
				},
			},
			Tools: []openai.ChatCompletionToolUnionParam{
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name: "Read",
					Description: openai.String("Read and return the contents of a file"),
					Parameters: openai.FunctionParameters{
						"type": "object",
						"properties": map[string]any{
							"file_path" : map[string]any{
								"type": "string",
								"description" : "The path of the file to read",
							},
						},
					},
					
				}),
			},
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(resp.Choices[0].Message.ToolCalls) != 0 {
		type Arguments struct {
			FilePath string `json:"file_path"`
		}
		var arguments Arguments
		err	:= json.Unmarshal([]byte(resp.Choices[0].Message.ToolCalls[0].Function.Arguments), &arguments)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(Read(arguments.FilePath))
	}

	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	// TODO: Uncomment the line below to pass the first stage
	// fmt.Print(resp.Choices[0].Message.Content)
}
