# Railway MCP Link Deployment

This repository is configured for easy deployment on Railway.

## Quick Deploy to Railway

1. Fork this repository
2. Connect your Railway account to GitHub
3. Create a new project in Railway
4. Select "Deploy from GitHub repo" and choose this repository
5. Railway will automatically detect Go and deploy your application

## Environment Variables

Railway will automatically set:
- `PORT`: The port your application should listen on (automatically handled)

## Local Development

```bash
# Install dependencies
go mod download

# Run locally
go run main.go serve --host 0.0.0.0 --port 8080
```

## Railway Configuration

This project includes:
- `railway.json`: Railway deployment configuration
- Modified `main.go`: Automatically detects Railway's PORT environment variable
- CORS enabled for web access

## Usage

After deployment, your MCP Link server will be available at your Railway-generated URL.

Example usage:
```
https://your-app.railway.app/sse?s=[OpenAPI-Spec-URL]&u=[API-Base-URL]&h=[Auth-Header]
```
