# Railway Deployment Guide

This guide explains how to deploy the MediaWiki MCP Server to Railway.

## Quick Deploy

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/mediawiki-mcp)

Or manually:

1. **Push to GitHub** (if not already done)
2. **Create a new project on Railway**: https://railway.app/new
3. **Select "Deploy from GitHub repo"**
4. **Select your repository**
5. Railway will automatically detect the Dockerfile and deploy

## Configuration

Railway automatically provides the `PORT` environment variable. The following optional environment variables can be configured in Railway's dashboard:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port (Railway sets this automatically) | `8080` |
| `MCP_RATE_LIMIT` | Requests per second per wiki | `10.0` |
| `MCP_CACHE_TTL` | Cache TTL for page content (seconds) | `300` |
| `MCP_CACHE_TTL_INFO` | Cache TTL for wiki info (seconds) | `3600` |
| `MCP_REQUEST_TIMEOUT` | HTTP request timeout (seconds) | `30` |
| `MCP_USER_AGENT` | Custom User-Agent string | `MediaWikiMCP/1.0` |

## Endpoints

After deployment, your service will be available at:

```
https://your-app.railway.app
```

- **MCP Endpoint**: `POST /mcp`
- **Health Check**: `GET /health`
- **Info**: `GET /`

## Test Your Deployment

```bash
# Health check
curl https://your-app.railway.app/health

# Initialize MCP
curl -X POST https://your-app.railway.app/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

# Get wiki info
curl -X POST https://your-app.railway.app/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"wiki_info","arguments":{"wiki_url":"https://en.wikipedia.org"}}}'
```

## Features

- ✅ **Stateless JSON responses** - No session management required
- ✅ **Auto-detecting API paths** - Works with any MediaWiki installation
- ✅ **Health checks** - Railway monitors `/health` endpoint
- ✅ **Graceful shutdown** - Handles SIGTERM/SIGINT properly
- ✅ **Non-root user** - Runs as unprivileged user (UID 1000)
- ✅ **Optimized build** - Multi-stage Docker build (~15MB final image)

## Monitoring

Railway provides automatic monitoring:
- CPU and memory usage
- Request logs
- Health check status
- Deployment history

## Scaling

Railway supports:
- **Horizontal scaling**: Add more replicas in Railway dashboard
- **Vertical scaling**: Adjust CPU/memory resources
- **Auto-scaling**: Configure based on metrics

## Cost Optimization

The server is lightweight:
- **CPU**: ~0.1 vCPU at idle, spikes during requests
- **Memory**: ~50MB at idle, ~100MB under load
- **Bandwidth**: Varies by usage

Railway's Hobby plan ($5/month) provides:
- $5 credit/month (includes generous free tier)
- Custom domains
- Automatic HTTPS
- GitHub deployments

## Troubleshooting

### Check logs
```bash
railway logs
```

### Health check failing
- Verify PORT environment variable is set correctly
- Check container logs for startup errors
- Ensure /health endpoint is accessible

### API requests timing out
- Increase `MCP_REQUEST_TIMEOUT` environment variable
- Check target wiki's availability
- Review rate limit settings

## Local Testing

Test the Docker image locally before deploying:

```bash
# Build
docker build -t mediawiki-mcp .

# Run
docker run -p 8080:8080 mediawiki-mcp

# Test
curl http://localhost:8080/health
```

## Support

- **Issues**: https://github.com/yourusername/mediawiki-mcp/issues
- **Railway Docs**: https://docs.railway.app
