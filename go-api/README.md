# Go API

HTTP API with MCP server support for resource discovery.

## Running

```bash
go run main.go
```

Server starts on `http://localhost:8080`

## Endpoints

### REST API

| Method | Path                  | Description                        |
|--------|-----------------------|------------------------------------|
| GET    | `/discovery/resources`| Returns list of available resources |

### MCP Server (SSE Transport)

| Method   | Path       | Description                          |
|----------|------------|--------------------------------------|
| GET      | `/mcp`     | Establish SSE connection for MCP     |
| POST     | `/mcp/*`   | Send MCP messages to session         |

#### Available MCP Tools

| Tool                 | Description                                      |
|----------------------|--------------------------------------------------|
| `discover_resources` | List all available resources in the discovery service |

## Example Response

```json
{
  "resources": [
    {
      "id": "1",
      "name": "Weather API",
      "description": "Real-time weather data for any location",
      "uri": "https://api.weather.example/v1"
    }
  ]
}
```
