# Python to Go Pattern Mapping Reference

## Type System

| Python | Go | Notes |
|--------|-----|-------|
| `str` | `string` | |
| `int` | `int` / `int64` | Use `int64` for IDs and timestamps |
| `float` | `float64` | |
| `bool` | `bool` | |
| `None` | pointer type or `error` | `*string` for nullable, `error` for failure |
| `bytes` | `[]byte` | |
| `list[T]` | `[]T` | |
| `dict[K, V]` | `map[K]V` | |
| `tuple[A, B]` | named struct or multiple returns | |
| `set[T]` | `map[T]struct{}` | |
| `Optional[T]` | `*T` | Pointer for optional fields |
| `Any` | `any` (= `interface{}`) | Avoid when possible, use typed interfaces |

## Class and Object Patterns

### Python class -> Go struct + interface

```python
# Python
class LLMProvider(ABC):
    @abstractmethod
    def chat(self, messages: list[Message]) -> Response:
        pass

    @abstractmethod
    def name(self) -> str:
        pass

class OpenAIProvider(LLMProvider):
    def __init__(self, api_key: str, model: str):
        self.api_key = api_key
        self.model = model

    def chat(self, messages):
        # implementation
        pass

    def name(self):
        return "openai"
```

```go
// Go - interface defined by consumer (internal/agent/)
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message) (*Response, error)
    Name() string
}

// Go - struct defined by provider (internal/llm/)
type OpenAIProvider struct {
    apiKey string
    model  string
    client *openai.Client
}

func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
    return &OpenAIProvider{
        apiKey: apiKey,
        model:  model,
        client: openai.NewClient(apiKey),
    }
}

func (p *OpenAIProvider) Chat(ctx context.Context, messages []Message) (*Response, error) {
    // implementation
}

func (p *OpenAIProvider) Name() string {
    return "openai"
}
```

### Inheritance -> Composition (Embedding)

```python
# Python
class BaseChannel:
    def send(self, msg): ...
    def receive(self): ...

class TelegramChannel(BaseChannel):
    def send(self, msg):
        # telegram-specific
        pass
```

```go
// Go - use embedding for shared behavior
type BaseChannel struct {
    name string
}

func (b *BaseChannel) ChannelName() string {
    return b.name
}

type TelegramChannel struct {
    BaseChannel  // embedding
    bot *telebot.Bot
}
```

## Async and Concurrency

### async/await -> goroutine + context

```python
# Python
async def process_messages(messages):
    tasks = [process_one(m) for m in messages]
    results = await asyncio.gather(*tasks)
    return results
```

```go
// Go
func processMessages(ctx context.Context, messages []Message) ([]Result, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]Result, len(messages))

    for i, msg := range messages {
        i, msg := i, msg
        g.Go(func() error {
            r, err := processOne(ctx, msg)
            if err != nil {
                return err
            }
            results[i] = r
            return nil
        })
    }

    if err := g.Wait(); err != nil {
        return nil, fmt.Errorf("process messages: %w", err)
    }
    return results, nil
}
```

### Python Queue -> Go channel

```python
# Python
queue = asyncio.Queue()
await queue.put(item)
item = await queue.get()
```

```go
// Go
ch := make(chan Item, bufferSize)
ch <- item      // send
item := <-ch    // receive
```

## Error Handling

### try/except -> if err != nil

```python
# Python
try:
    result = provider.chat(messages)
except APIError as e:
    logger.error(f"Chat failed: {e}")
    raise ServiceError("chat failed") from e
```

```go
// Go
result, err := provider.Chat(ctx, messages)
if err != nil {
    return nil, fmt.Errorf("chat with %s: %w", provider.Name(), err)
}
```

### Context Manager -> defer

```python
# Python
with open("file.txt") as f:
    data = f.read()
```

```go
// Go
f, err := os.Open("file.txt")
if err != nil {
    return err
}
defer f.Close()
data, err := io.ReadAll(f)
```

## Decorator -> Middleware / Wrapper

```python
# Python
def retry(max_attempts=3):
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            for attempt in range(max_attempts):
                try:
                    return await func(*args, **kwargs)
                except Exception:
                    if attempt == max_attempts - 1:
                        raise
        return wrapper
    return decorator

@retry(max_attempts=3)
async def call_api(): ...
```

```go
// Go - middleware function
func WithRetry(maxAttempts int, fn func(ctx context.Context) error) func(ctx context.Context) error {
    return func(ctx context.Context) error {
        var lastErr error
        for i := 0; i < maxAttempts; i++ {
            if err := fn(ctx); err != nil {
                lastErr = err
                continue
            }
            return nil
        }
        return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
    }
}
```

## Configuration

### Pydantic Model -> Go struct + viper

```python
# Python (Pydantic)
class Config(BaseModel):
    host: str = "0.0.0.0"
    port: int = 8088
    model_provider: str = "openai"
```

```go
// Go
type Config struct {
    Host          string `mapstructure:"host" yaml:"host"`
    Port          int    `mapstructure:"port" yaml:"port"`
    ModelProvider string `mapstructure:"model_provider" yaml:"model_provider"`
}

func LoadConfig(path string) (*Config, error) {
    v := viper.New()
    v.SetConfigFile(path)
    v.SetDefault("host", "0.0.0.0")
    v.SetDefault("port", 8088)
    v.SetDefault("model_provider", "openai")

    if err := v.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    return &cfg, nil
}
```

## Registry Pattern

### Python dict-based registry -> Go map + Register function

```python
# Python
CHANNEL_REGISTRY = {}

def register_channel(name):
    def decorator(cls):
        CHANNEL_REGISTRY[name] = cls
        return cls
    return decorator

@register_channel("telegram")
class TelegramChannel: ...
```

```go
// Go
type ChannelFactory func(cfg ChannelConfig) (Channel, error)

var channelRegistry = map[string]ChannelFactory{}

func RegisterChannel(name string, factory ChannelFactory) {
    channelRegistry[name] = factory
}

func init() {
    RegisterChannel("telegram", NewTelegramChannel)
}
```

## Generator / Yield -> Channel or Iterator

```python
# Python
def stream_response(prompt):
    for chunk in llm.stream(prompt):
        yield chunk.text
```

```go
// Go - channel-based streaming
func StreamResponse(ctx context.Context, prompt string) <-chan string {
    ch := make(chan string)
    go func() {
        defer close(ch)
        stream := llm.Stream(ctx, prompt)
        for {
            chunk, err := stream.Recv()
            if err != nil {
                return
            }
            select {
            case ch <- chunk.Text:
            case <-ctx.Done():
                return
            }
        }
    }()
    return ch
}
```
