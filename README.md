# Nginx Config Viewer

A single binary Golang nginx configuration webviewer with React frontend, Monaco Editor, live reload, and integrated documentation tooltips.

## Features

- **React UI**: Readonly Monaco Editor and nginx syntax highlighting via [jaywcjlove/nginx-editor/website](https://github.com/jaywcjlove/nginx-editor/tree/main/website)
- **Live reload**: Automatically refreshes when config files change (via Server-Sent Events)  
- **Built-in tooltips**: Hover over nginx directives to see official documentation
- **API endpoints**: `/raw` for plain text access, `/events` for live updates
- **Single binary**: Complete React frontend embedded in Go binary (~30MB)
- **Security hardened**: Read-only access, CSP headers, minimal attack surface

## Quick Start

### Download Pre-built Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/vincentdesmet/nginx-config-viewer/releases):

```bash
# Linux AMD64
wget https://github.com/vincentdesmet/nginx-config-viewer/releases/latest/download/nginx-config-viewer_Linux_x86_64.tar.gz

# Linux ARM64  
wget https://github.com/vincentdesmet/nginx-config-viewer/releases/latest/download/nginx-config-viewer_Linux_arm64.tar.gz

# macOS Intel
wget https://github.com/vincentdesmet/nginx-config-viewer/releases/latest/download/nginx-config-viewer_Darwin_x86_64.tar.gz

# macOS Apple Silicon
wget https://github.com/vincentdesmet/nginx-config-viewer/releases/latest/download/nginx-config-viewer_Darwin_arm64.tar.gz
```

Extract and run:
```bash
tar -xzf nginx-config-viewer_*.tar.gz
./nginx-config-viewer -addr :8080 -path /etc/nginx/nginx.conf
```

### Build from Source

#### Prerequisites
- Go 1.21+
- Node.js 20.9.0+
- pnpm 9+
- GoReleaser (for snapshot builds): `brew install goreleaser` or see [installation guide](https://goreleaser.com/install/)

#### Build Commands
```bash
# Using Makefile (recommended)
make build

# Or manually
pnpm install --dir website
go generate ./...
go build -o nginx-config-viewer
```

#### Development
```bash
# Run development server
make dev

# Create local snapshot builds for all platforms  
make snapshot

# Build specific platform
make build-linux-arm64
```

## Usage

```bash
./nginx-config-viewer [options]

Options:
  -addr string
        listen address (default ":8080")
  -cors
        allow CORS on /raw (off by default)  
  -path string
        nginx.conf path (default "/etc/nginx/nginx.conf")
  -version
        show version information
```

Visit http://localhost:8080 to view your nginx config with syntax highlighting and live reload.

## API Endpoints

- `/` - React web interface with Monaco Editor
- `/raw` - Plain text nginx.conf for curl/scripts  
- `/events` - Server-Sent Events for live reload
- `/static/*` - React app assets (JS/CSS/media)

## Production Deployment

### Systemd Service (Linux)

1. **Install binary:**
   ```bash
   sudo install -m755 nginx-config-viewer /usr/local/bin/
   ```

2. **Create systemd service:**
   ```bash
   sudo tee /etc/systemd/system/nginx-config-viewer.service > /dev/null <<EOF
   [Unit]
   Description=nginx.conf viewer with SSE reload  
   After=network.target

   [Service]
   ExecStart=/usr/local/bin/nginx-config-viewer -addr :8080 -path /etc/nginx/nginx.conf
   User=www-data
   Group=www-data
   Restart=on-failure
   NoNewPrivileges=true
   ProtectSystem=strict
   ProtectHome=true
   ReadOnlyPaths=/etc/nginx

   [Install]
   WantedBy=multi-user.target
   EOF
   ```

3. **Enable and start:**
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable --now nginx-config-viewer
   sudo systemctl status nginx-config-viewer
   ```

## Remote Access via AWS SSM

### Secure Access to EC2 Instances

Access your nginx config viewer on EC2 instances **without opening inbound ports or using SSH** via AWS Systems Manager Session Manager port forwarding.

#### Prerequisites

- EC2 instance with SSM Agent installed
- Instance IAM role with `AmazonSSMManagedInstanceCore` policy
- AWS CLI v2 with Session Manager plugin installed
- Local AWS credentials configured

#### Basic Port Forwarding

Forward a local port to the nginx-config-viewer running on EC2:

```bash
# Forward local port 18080 to instance port 8080
aws ssm start-session \
  --target i-0123456789abcdef0 \
  --document-name AWS-StartPortForwardingSession \
  --parameters "portNumber=8080,localPortNumber=18080"
```

Then open: **http://localhost:18080/**

#### Advanced: Remote Host Forwarding

If nginx-config-viewer runs on a different interface (e.g., Docker):

```bash
# Forward to specific host/port on the instance
aws ssm start-session \
  --target i-0123456789abcdef0 \
  --document-name AWS-StartPortForwardingSessionToRemoteHost \
  --parameters "host=127.0.0.1,portNumber=8080,localPortNumber=18080"
```

#### Quick Setup Script

Create a convenience script for automatic tunneling:

```bash
#!/bin/bash
# tunnel-nginx-viewer.sh
INSTANCE_ID="${1:-i-0123456789abcdef0}"
LOCAL_PORT="${2:-18080}"
REMOTE_PORT="${3:-8080}"

echo "🚀 Starting SSM tunnel to $INSTANCE_ID"
echo "📡 Local: http://localhost:$LOCAL_PORT"
echo "🔗 Remote: $INSTANCE_ID:$REMOTE_PORT"
echo "Press Ctrl+C to stop..."

aws ssm start-session \
  --target "$INSTANCE_ID" \
  --document-name AWS-StartPortForwardingSession \
  --parameters "portNumber=$REMOTE_PORT,localPortNumber=$LOCAL_PORT"
```

Usage:
```bash
chmod +x tunnel-nginx-viewer.sh
./tunnel-nginx-viewer.sh i-0abc123def456789 18080 8080
```

#### Benefits

- **No inbound security group rules** required
- **No public IP** or bastion host needed  
- **Full SSE support** for live reload functionality
- **IAM-controlled access** with session logging
- **Encrypted tunnel** through AWS infrastructure

#### Troubleshooting

- **Instance not found**: Verify SSM Agent is installed and instance has proper IAM role
- **Connection timeout**: Check instance is running and nginx-config-viewer is bound to correct interface
- **SSE not working**: Ensure steady browser activity to prevent idle timeouts

## Development

### Release Process

This project uses automated releases via [release-please](https://github.com/googleapis/release-please) and [GoReleaser](https://goreleaser.com/):

1. **Commit changes** to `main` branch
2. **Release PR created** automatically with changelog
3. **Merge release PR** to create GitHub release + git tag
4. **Binaries built** automatically for all platforms and attached to release

### Manual Release Testing

```bash
# Test local snapshot build
make snapshot

# Check generated binaries
ls -la dist/
```

## Architecture

```
Single Binary (30MB)
├── Go HTTP Server
│   ├── /raw (nginx config endpoint)
│   ├── /events (Server-Sent Events)  
│   └── /* (React SPA routing)
└── Embedded React App
    ├── Monaco Editor (syntax highlighting)
    ├── Built-in nginx directive tooltips  
    ├── Dark/light theme switching
    └── Live reload via SSE
```

**Key Components:**
- **Backend**: Go with fsnotify for file watching, embedded filesystem for assets
- **Frontend**: React + Monaco Editor with nginx language plugin  
- **Build**: Go generate triggers pnpm build, embeds React in binary
- **Deployment**: Single static binary, no external dependencies