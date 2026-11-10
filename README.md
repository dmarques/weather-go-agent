# Weather Agent in Go

A minimal HTTP agent that retrieves weather information using a `get_weather` tool and the OpenAI Responses API.

## Included security measures

- Does not expose the OpenAI API key.
- Requires `X-Agent-Token` to access `/chat`.
- Limits request bodies to 4 KB.
- Limits requests to 20 per IP address per minute.
- Limits concurrent requests to 20.
- Limits each request to 2 tool calls.
- Uses network timeouts.
- Allows only the weather tool.
- Does not retain history between requests.
- Does not log message content.

## Run locally

```bash
cp .env.example .env
export OPENAI_API_KEY='your-api-key'
export AGENT_API_TOKEN='a-long-random-token'
go run ./cmd/api
```

The service listens on `127.0.0.1:8080` by default.

Test it:

```bash
curl -s http://127.0.0.1:8080/healthz
```

Response example:

```json
{
  "status": "ok"
}
```

Test it:

```bash
curl -s http://127.0.0.1:8080/chat \
  -H 'Content-Type: application/json' \
  -H 'X-Agent-Token: a-long-random-token' \
  -d '{"message":"Will it rain in São Paulo today?"}'
```

Response example:

```json
{
  "answer": "São Paulo has a 19% chance of rain today, so rain is possible but not very likely.",
  "usage": {
    "input_tokens": 201,
    "output_tokens": 25,
    "total_tokens": 226
  }
}
```

## VPS

Place an HTTPS reverse proxy in front of the process and keep `LISTEN_ADDR=127.0.0.1:8080`. Do not expose the port directly to the internet. The recommended next step is to create a systemd service and configure additional limits in the proxy.
