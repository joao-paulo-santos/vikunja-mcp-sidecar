# Vikunja MCP Sidecar

An MCP (Model Context Protocol) server that runs as a Docker sidecar alongside [Vikunja](https://vikunja.io), letting AI agents manage your projects and tasks through the Vikunja API.

## Features

- Full project CRUD (create, read, update, delete)
- Full task CRUD with priorities, due dates, and labels
- Task comments
- Label management
- Namespace listing
- SSE transport for network-accessible AI agent connections
- Stdio transport for local development
- Single static binary, minimal Docker image

## Tools

| Tool | Description |
|------|-------------|
| `list_namespaces` | List all namespaces with their projects |
| `list_projects` | List all projects |
| `get_project` | Get project details by ID |
| `create_project` | Create a new project in a namespace |
| `update_project` | Update a project's title, description, or archive status |
| `delete_project` | Delete a project and all its tasks |
| `list_tasks` | List tasks, optionally filtered by project |
| `get_task` | Get full task details by ID |
| `create_task` | Create a task with title, description, due date, priority, and labels |
| `update_task` | Update task properties, mark as done, move between projects, set labels |
| `delete_task` | Delete a task |
| `list_task_comments` | List comments on a task |
| `add_task_comment` | Add a comment to a task |
| `list_task_assignees` | List users assigned to a task |
| `add_task_assignee` | Assign a user to a task |
| `remove_task_assignee` | Remove a user from a task |
| `list_labels` | List all labels |
| `create_label` | Create a new label with title and color |
| `add_task_label` | Add a label to a task |
| `remove_task_label` | Remove a label from a task |
| `list_views` | List all views (list, gantt, table, kanban) for a project |
| `list_buckets` | List Kanban buckets for a project view |
| `move_task_to_bucket` | Move a task to a different Kanban bucket |

## Quick Start

### 1. Generate an API token

In Vikunja, go to **Settings > API Tokens** and create a new token.

### 2. Create a `.env` file

```bash
cp .env.example .env
```

Edit `.env` and paste your token:

```
VIKUNJA_API_TOKEN=tk_your_token_here
```

### 3. Add the MCP service to your docker-compose.yml

The `docker-compose.yml` in this repo is tailored to a specific nginx reverse proxy setup. For a standard Vikunja deployment, add just the MCP service:

```yaml
services:
  vikunja:
    image: vikunja/vikunja
    environment:
      VIKUNJA_SERVICE_PUBLICURL: http://localhost:3456/
      VIKUNJA_DATABASE_PATH: /db/vikunja.db
    ports:
      - 3456:3456
    volumes:
      - ./files:/app/vikunja/files
      - ./db:/db

  mcp:
    build: ./mcp-server
    environment:
      VIKUNJA_API_URL: http://vikunja:3456
      VIKUNJA_API_TOKEN: ${VIKUNJA_API_TOKEN}
    ports:
      - 3666:3666
    depends_on:
      - vikunja
    restart: unless-stopped
```

### 4. Start the services

```bash
docker compose up -d
```

### 5. Connect your AI agent

The MCP server exposes an SSE endpoint at `http://localhost:3666/sse`.

**Claude Desktop** (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "vikunja": {
      "url": "http://localhost:3666/sse"
    }
  }
}
```

**opencode** (`.opencode.json`):

```json
{
  "mcp": {
    "vikunja": {
      "url": "http://localhost:3666/sse"
    }
  }
}
```

**Any MCP client** using the SSE protocol can connect to:
- SSE endpoint: `http://localhost:3666/sse`
- Message endpoint: `http://localhost:3666/message`
- Health check: `http://localhost:3666/health`

## Local Development (stdio)

For running outside Docker with stdio transport:

```bash
cd mcp-server
VIKUNJA_API_URL=http://localhost:3456 \
VIKUNJA_API_TOKEN=your_token \
go run . -listen ""
```

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `VIKUNJA_API_URL` | Yes | Vikunja API base URL (e.g. `http://vikunja:3456`) |
| `VIKUNJA_API_TOKEN` | Yes | Vikunja API token |

| Flag | Default | Description |
|------|---------|-------------|
| `-listen` | `:3666` | Listen address for SSE. Pass empty string `""` for stdio transport |

## Building

```bash
cd mcp-server
docker build -t vikunja-mcp .
```

Or natively:

```bash
cd mcp-server
go build -o vikunja-mcp .
```
