# LuoxueAPI Docker Image

LuoxueAPI is an AI API Gateway Platform for distributing and managing AI product subscription API quotas.

> The image shown below is the public upstream compatibility image. This private Fork's GHCR package is not anonymously pullable; build the repository locally or authenticate to its private registry to deploy LuoxueAPI-specific changes.

## Quick Start

```bash
docker run -d \
  --name sub2api \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@host:5432/sub2api" \
  -e REDIS_URL="redis://host:6379" \
  ghcr.io/3246147496lxt-dotcom/luoxueapi:1.0.7
```

## Docker Compose

```yaml
version: '3.8'

services:
  sub2api:
    image: ghcr.io/3246147496lxt-dotcom/luoxueapi:1.0.7
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://postgres:postgres@db:5432/sub2api?sslmode=disable
      - REDIS_URL=redis://redis:6379
    depends_on:
      - db
      - redis

  db:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=postgres
      - POSTGRES_DB=sub2api
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Yes | - |
| `REDIS_URL` | Redis connection string | Yes | - |
| `PORT` | Server port | No | `8080` |
| `GIN_MODE` | Gin framework mode (`debug`/`release`) | No | `release` |

## Supported Architectures

- `linux/amd64`
- `linux/arm64`

## Tags

- `latest` - Latest stable release
- `x.y.z` - Specific version
- `x.y` - Latest patch of minor version
- `x` - Latest minor of major version

## Links

- [GitHub Repository](https://github.com/3246147496lxt-dotcom/luoxueapi)
- [Documentation](https://github.com/3246147496lxt-dotcom/luoxueapi#readme)
