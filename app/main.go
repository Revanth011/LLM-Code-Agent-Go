package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

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
type ReadArguments struct {
	FilePath string `json:"filePath"`
}
type WriteArguments struct {
	FilePath string `json:"filePath"`
	Data 	 string `json:"data"`
}

type BashArguments struct {
	Command string `json:"command"`
}

func Read(filePath string) string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	return string(data)
}

func Write(filePath string,  data []byte) string{
	err := os.WriteFile(filePath, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return "file written successfully"
}

func Bash(command string) string {
	cmd := exec.Command("sh", "-c", command)

	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	return string(output)
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
	var messages []openai.ChatCompletionMessageParamUnion
	var tools []openai.ChatCompletionToolUnionParam

	messages = append(messages, openai.ChatCompletionMessageParamUnion{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(prompt),
						},
					},})

	tools = append(tools, 
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name: "Read",
					Description: openai.String("Read and return the contents of a file"),
					Parameters: openai.FunctionParameters{
						"type": "object",
						"properties": map[string]any{
							"filePath" : map[string]any{
								"type": "string",
								"description" : "The path of the file to read",
							},
						},
					},
					
				}),
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name: "Write",
					Description: openai.String("Write content to the file"),
					Parameters: openai.FunctionParameters{
						"type": "object",
						"properties": map[string]any{
							"filePath" : map[string]any{
								"type": "string",
								"description" : "The path of the file to write",
							},
							"data" : map[string]any{
								"type": "string",
								"description" : "Data to write to the file",
							},
						},
					},
					
				}),
				openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
					Name: "Bash",
					Description: openai.String("Execute bash commands"),
					Parameters: openai.FunctionParameters{
						"type": "object",
						"properties": map[string]any{
							"command" : map[string]any{
								"type": "string",
								"description" : "bash command to execute",
							},
						},
					},
					
				}),
			)
	
	resp, err := client.Chat.Completions.New(context.Background(),
		openai.ChatCompletionNewParams{
			Model: "anthropic/claude-haiku-4.5",
			Messages: messages,
			Tools: tools,
		},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	for len(resp.Choices[0].Message.ToolCalls) != 0 {
		messages = append(messages, resp.Choices[0].Message.ToParam())
		

		var toolName string = resp.Choices[0].Message.ToolCalls[0].Function.Name
		switch toolName {
			case "Read":
				var arguments ReadArguments
				err	:= json.Unmarshal([]byte(resp.Choices[0].Message.ToolCalls[0].Function.Arguments), &arguments)
				if err != nil {
					log.Fatal(err)
				}
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
							OfTool: &openai.ChatCompletionToolMessageParam{
								ToolCallID: resp.Choices[0].Message.ToolCalls[0].ID,
								Content:  openai.ChatCompletionToolMessageParamContentUnion {
								OfString:openai.String(Read(arguments.FilePath))},
							},})

			case "Write":
				var arguments WriteArguments
				err	:= json.Unmarshal([]byte(resp.Choices[0].Message.ToolCalls[0].Function.Arguments), &arguments)
				if err != nil {
					log.Fatal(err)
				}
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
							OfTool: &openai.ChatCompletionToolMessageParam{
								ToolCallID: resp.Choices[0].Message.ToolCalls[0].ID,
								Content:  openai.ChatCompletionToolMessageParamContentUnion {
								OfString:openai.String(Write(arguments.FilePath, []byte(arguments.Data)))},
							},})

			case "Bash":
				var arguments BashArguments
				err	:= json.Unmarshal([]byte(resp.Choices[0].Message.ToolCalls[0].Function.Arguments), &arguments)
				if err != nil {
					log.Fatal(err)
				}
				messages = append(messages, openai.ChatCompletionMessageParamUnion{
							OfTool: &openai.ChatCompletionToolMessageParam{
								ToolCallID: resp.Choices[0].Message.ToolCalls[0].ID,
								Content:  openai.ChatCompletionToolMessageParamContentUnion {
								OfString:openai.String(Bash(arguments.Command))},
							},})
			}
			resp, err = client.Chat.Completions.New(context.Background(),
			openai.ChatCompletionNewParams{
				Model: "anthropic/claude-haiku-4.5",
				Messages: messages,
				Tools: tools,
			},
		)
	}
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Fprintln(os.Stderr, "Logs from your program will appear here!")

	// TODO: Uncomment the line below to pass the first stage
	fmt.Print(resp.Choices[0].Message.Content)
}
