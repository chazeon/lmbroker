# LMBroker

A configurable LLM broker written in Go that acts as intelligent middleware between clients and various LLM providers (OpenAI, Anthropic, etc.). LMBroker provides automatic model-based routing, bidirectional API translation, and optimized passthrough when client and backend formats match.

## ✨ Features

- **Model-Based Routing**: Automatic backend selection based on model names in requests
- **Multi-Provider Support**: OpenAI, Anthropic, and any OpenAI-compatible APIs (Ollama, etc.)
- **Smart Translation**: Bidirectional conversion between API formats when needed
- **Optimized Passthrough**: Direct streaming when client/backend formats match
- **Tool/Function Calling**: Full support with automatic format conversion
- **Production Ready**: Health checks, Prometheus metrics, structured logging
- **Model Aliasing**: Map any model name to any provider (e.g., `gpt-4-local` → Ollama)

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or higher

### Installation

```bash
git clone https://github.com/chazoen/lmbroker.git
cd lmbroker
go mod tidy
go build -o lmbroker ./cmd/lmbroker
```

### Configuration

Create `config.toml` with your model mappings:

```toml
log_level = "info"

[server]
  host = "localhost"
  port = 8080

# Map model names to providers
[[models]]
  alias = "claude-3-haiku-20240307"  # Model name clients request
  target = { url = "https://api.anthropic.com/v1/", model = "claude-3-haiku-20240307", api_key = "sk-ant-..." }
  type = "anthropic"                 # Provider API format

[[models]]
  alias = "gpt-4"                   # Model name clients request
  target = { url = "https://api.openai.com/v1/", model = "gpt-4", api_key = "sk-..." }
  type = "openai"                   # Provider API format

# Model aliasing - use local Ollama but client thinks it's GPT-4
[[models]]
  alias = "gpt-4-local"             # Clients request this
  target = { url = "http://localhost:11434/v1/", model = "llama3.1" }
  type = "openai"

# Environment variable support - use env: prefix
[[models]]
  alias = "gpt-4-secure"
  target = { url = "https://api.openai.com/v1/", model = "gpt-4", api_key = "env:OPENAI_API_KEY" }
  type = "openai"
```

**Security Note:** Use `api_key = "env:VARIABLE_NAME"` to load API keys from environment variables in production.

### Run

```bash
# Set environment variables for API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

./lmbroker
```

Server starts on `http://localhost:8080`

## 📖 Usage

LMBroker automatically routes requests based on the model name in the request body. No special headers required!

### OpenAI Format Client

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

### Anthropic Format Client

```bash
curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-haiku-20240307",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 100
  }'
```

### Embeddings

```bash
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -d '{
    "model": "text-embedding-ada-002",
    "input": "Hello world"
  }'
```

### Cross-Provider Translation

Use OpenAI client with Anthropic backend automatically:

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-haiku-20240307",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 🏗️ How It Works

1. **Route Detection**: LMBroker identifies client format from URL path
2. **Model Extraction**: Extracts model name from request body  
3. **Backend Lookup**: Finds configured provider for that model
4. **Smart Execution**: 
   - **Passthrough**: Direct streaming when formats match (optimal)
   - **Translation**: 4-step conversion through unified internal format

## 🔌 API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/chat/completions` | OpenAI-format chat completions |
| `POST` | `/v1/messages` | Anthropic-format messages |
| `POST` | `/v1/embeddings` | OpenAI-format embeddings |
| `GET` | `/health` | Health check |
| `GET` | `/metrics` | Prometheus metrics |

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Test specific package
go test ./internal/broker/

# Verbose output
go test -v ./...
```

## 📊 Monitoring

- **Health Check**: `GET /health`
- **Metrics**: `GET /metrics` (Prometheus format)
- **Structured Logging**: JSON format with configurable levels

## 🏛️ Architecture

- **Broker**: Main orchestrator with operation-specific handlers
- **Adapters**: Provider-specific translation logic (OpenAI/Anthropic)
- **Workflows**: Execution patterns (passthrough vs translation)
- **Unified Model**: Internal format for seamless provider translation

## 📋 Development

```bash
# Build
go build -o lmbroker ./cmd/lmbroker

# Run tests
go test ./...

# Update dependencies  
go mod tidy

# Development run
go run ./cmd/lmbroker
```
