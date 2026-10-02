package einocodex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/schema/openai"
)

const codexBaseURL = "https://chatgpt.com/backend-api/codex"

// NewModel creates a codex subscription compliant responses model.
func NewModel(ctx context.Context, conf *ResponsesConfig) (*ResponsesModel, error) {
	am := newAuthManager()
	httpClient := &http.Client{
		Transport: &authRoundTripper{
			base:        http.DefaultTransport,
			authManager: am,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	var store bool
	m, err := agenticopenai.NewResponsesModel(ctx, &agenticopenai.ResponsesConfig{
		HTTPClient: httpClient,
		Store:      &store,
		BaseURL:    codexBaseURL,
	})
	if err != nil {
		return nil, err
	}
	return &ResponsesModel{
		ResponsesModel: m,
	}, nil
}

type ResponsesConfig struct {
	Model string
}

type ResponsesModel struct {
	*agenticopenai.ResponsesModel
}

// Generate overrides the original due to how the codex endpoint only allows for streaming requests and does not support generate. This buffers the stream and returns the chunks as a single message.
func (rm *ResponsesModel) Generate(ctx context.Context, input []*schema.AgenticMessage, opts ...model.Option) (*schema.AgenticMessage, error) {
	stream, err := rm.Stream(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var chunks []*schema.AgenticMessage
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	msg, err := schema.ConcatAgenticMessages(chunks)
	if err != nil {
		return nil, err
	}
	if msg.ResponseMeta == nil || msg.ResponseMeta.OpenAIExtension == nil || msg.ResponseMeta.OpenAIExtension.Status != openai.ResponseStatusCompleted {
		return nil, errors.New("message generation was interrupted")
	}
	return msg, nil
}

type authRoundTripper struct {
	base        http.RoundTripper
	authManager *authManager
}

func (art *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	u := req.URL
	if u.Scheme != "https" || u.Host != "chatgpt.com" || u.User != nil || u.EscapedPath() != "/backend-api/codex/responses" {
		return nil, fmt.Errorf("unsafe url: %q to send codex credentials", u)
	}

	if err := art.authManager.ensureSession(); err != nil {
		return nil, err
	}
	token := art.authManager.AccessToken()
	if token == "" {
		return nil, errors.New("empty access token")
	}

	areq := req.Clone(req.Context())
	areq.Header.Set("Authorization", "Bearer "+token)
	return art.base.RoundTrip(areq)
}
