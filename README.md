# einocodex

Adapter for eino's openai ResponsesModel. This allows you to utilise your codex subscription instead of using a openai api key.

In other words this allows you to test your eino agents locally using codex subscription.

## Quickstart

Prerequisites:

1. Go v1.27
2. Logged in to codex at least once
3. Optional: set CODEX_HOME

Install with:
```sh
go get github.com/brandonyeoxg/einocodex
```

```go
model, err := einocodex.NewModel(context.Background(), &einocodex.ResponsesConfig{
    Model: "OPENAI_MODEL_ID",
})
if err != nil {
    ...
}

userMsg := schema.UserAgenticMessage("Say hello world in one sentence.")
stream, err := model.Stream(context.Background(), []*schema.AgenticMessage{
    userMsg,
})
if err != nil {
    ...
}
defer stream.Close()

for {
    d, err := stream.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    ...
}
```

## Examples

There are examples that you can reference and run:

1. stream - [main.go](./examples/stream/main.go)
2. generate - [main.go](./examples/generate/main.go)
