package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dmarques/weather-go-agent/internal/application/chat"
)

type OpenAIClient struct {
	httpClient *http.Client
	apiKey     string
}

// NewOpenAIClient creates a new OpenAIClient with the provided HTTP client and API key.
func NewOpenAIClient(httpClient *http.Client, apiKey string) *OpenAIClient {
	return &OpenAIClient{httpClient: httpClient, apiKey: apiKey}
}

type responseEnvelope struct {
	Output []json.RawMessage `json:"output"`
	Usage  chat.Usage        `json:"usage"`
}

// CreateResponse sends a request to the OpenAI API to generate a response based on the provided instructions and input, and returns the structured response.
func (c *OpenAIClient) CreateResponse(ctx context.Context, req chat.ResponseRequest) (chat.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return chat.Response{}, err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.openai.com/v1/responses",
		bytes.NewReader(body),
	)
	if err != nil {
		return chat.Response{}, err
	}
	request.Header.Set("Authorization", "Bearer "+c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return chat.Response{}, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return chat.Response{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return chat.Response{}, fmt.Errorf("openai returned status %d", response.StatusCode)
	}

	var decoded responseEnvelope
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return chat.Response{}, err
	}

	output, err := decodeOutput(decoded.Output)
	if err != nil {
		return chat.Response{}, err
	}
	return chat.Response{Output: output, Usage: decoded.Usage}, nil
}

// decodeOutput processes the raw JSON output from the OpenAI API and converts it into a slice of structured OutputItem objects.
func decodeOutput(items []json.RawMessage) ([]chat.OutputItem, error) {
	output := make([]chat.OutputItem, 0, len(items))
	for _, raw := range items {
		var base struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			return nil, err
		}

		item := chat.OutputItem{
			Type: base.Type,
			Raw:  append(json.RawMessage(nil), raw...),
		}
		switch base.Type {
		case "function_call":
			var call struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
				CallID    string `json:"call_id"`
			}
			if err := json.Unmarshal(raw, &call); err != nil {
				return nil, err
			}
			item.Name = call.Name
			item.Arguments = call.Arguments
			item.CallID = call.CallID
		case "message":
			var message struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			}
			if err := json.Unmarshal(raw, &message); err != nil {
				return nil, err
			}
			var text strings.Builder
			for _, part := range message.Content {
				if part.Type == "output_text" {
					text.WriteString(part.Text)
				}
			}
			item.Text = text.String()
		}
		output = append(output, item)
	}
	return output, nil
}
