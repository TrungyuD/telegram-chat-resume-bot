# Deployment Guide

## Quick Start

### Prerequisites

| Requirement | Details |
|-------------|---------|
| Go | 1.25.4+ |
| Claude CLI | Installed and authenticated (see [Claude CLI docs](https://docs.anthropic.com/en/docs/claude-code)) |
| Telegram Bot Token | Created via [@BotFather](https://t.me/BotFather) |
| Admin Telegram ID | Your numeric Telegram user ID (find via [@userinfobot](https://t.me/userinfobot)) |
| Server/Machine | Linux, macOS, or Windows with 512MB+ RAM |

### Installation (5 minutes)

```bash
# Clone the repository
git clone https://github.com/TrungyuD/telegram-chat-resume-bot.git
cd telegram-chat-resume-bot

# Copy configuration template
cp .env.example .env

# Edit .env with your values
nano .env  # or use your preferred editor
# Set: TELEGRAM_BOT_TOKEN, ADMIN_TELEGRAM_IDS

# Install dependencies
go mod tidy

# Build the binary
go build -o dist/telegram-claude-bot ./cmd/bot/

# Run!
./dist/telegram-claude-bot
```

**Expected output:**
```
2025/03/16 14:23:45 Bot started successfully
2025/03/16 14:23:45 Dashboard server running on http://localhost:3000
```

---

## Environment Configuration

### Required Variables

```bash
# Telegram
TELEGRAM_BOT_TOKEN=123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11

# Admin authorization
ADMIN_TELEGRAM_IDS=123456789,987654321
```

### Optional Variables (Sensible Defaults Provided)

```bash
# Claude AI defaults
CLAUDE_DEFAULT_MODEL=claude-sonnet-4-6          # sonnet/opus/haiku
CLAUDE_DEFAULT_EFFORT=high                      # low/medium/high/max
CLAUDE_DEFAULT_THINKING=adaptive                # on/off/adaptive

# Query execution
CLAUDE_CLI_TIMEOUT_MS=360000                    # 6 minutes
CLAUDE_CONTEXT_MESSAGES=10                      # messages to include

# Concurrency
MAX_CONCURRENT_CLI_PROCESSES=3                  # system-wide limit
RATE_LIMIT_REQUESTS_PER_MINUTE=10               # per-user limit

# HTTP Dashboard
WEB_PORT=3000
ADMIN_API_KEY=your-secret-key-here              # for /api endpoints

# Session compaction
COMPACT_ENABLED=true
COMPACT_THRESHOLD=20                            # messages before compaction
COMPACT_KEEP_RECENT=6                           # messages to keep

# File access restrictions
ALLOWED_WORKING_DIRS=/home/user/projects,/tmp  # comma-separated, empty=any
```

### Configuration Files

**`.env` file** (local development):
```bash
# .env
TELEGRAM_BOT_TOKEN=your_token
ADMIN_TELEGRAM_IDS=123456789
CLAUDE_DEFAULT_MODEL=claude-sonnet-4-6
WEB_PORT=3000
```

**`data/config.json`** (runtime override, no restart needed):
```json
{
  "claude_default_model": "claude-opus-4-6",
  "rate_limit_requests_per_minute": "20",
  "web_port": "3001"
}
```

**Priority order:**
1. `.env` file (highest priority)
2. `data/config.json` (runtime override)
3. Hardcoded defaults (lowest priority)

---

## Deployment Scenarios

### Scenario 1: Local Development

**Setup:**
```bash
go run ./cmd/bot/
```

**What happens:**
- Reads `.env` file
- Starts bot and dashboard
- Creates `data/` directory automatically
- Ready for testing

**Stop:** Press `Ctrl+C`

---

### Scenario 2: Single Binary on Linux Server

**Step 1: Build on development machine**
```bash
GOOS=linux GOARCH=amd64 go build -o telegram-claude-bot ./cmd/bot/
```

**Step 2: Upload to server**
```bash
scp telegram-claude-bot user@server:/home/botuser/app/
scp .env user@server:/home/botuser/app/
ssh user@server "chmod +x /home/botuser/app/telegram-claude-bot"
```

**Step 3: Run on server**
```bash
cd /home/botuser/app
./telegram-claude-bot
```

**For 24/7 operation: Use systemd service (see below)**

---

### Scenario 3: Systemd Service (Linux)

**Step 1: Create service file**

Create `/etc/systemd/system/telegram-claude-bot.service`:

```ini
[Unit]
Description=Telegram Claude Bot
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=botuser
Group=botuser
WorkingDirectory=/opt/telegram-claude-bot

# Load environment variables
EnvironmentFile=/opt/telegram-claude-bot/.env

# Start the bot
ExecStart=/opt/telegram-claude-bot/telegram-claude-bot

# Restart policy
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

**Step 2: Prepare directories**

```bash
sudo mkdir -p /opt/telegram-claude-bot
sudo useradd -m -s /bin/bash botuser || true
sudo chown -R botuser:botuser /opt/telegram-claude-bot

# Copy binary and config
sudo cp telegram-claude-bot /opt/telegram-claude-bot/
sudo cp .env /opt/telegram-claude-bot/
sudo chown botuser:botuser /opt/telegram-claude-bot/{telegram-claude-bot,.env}
sudo chmod 600 /opt/telegram-claude-bot/.env
```

**Step 3: Enable and start service**

```bash
sudo systemctl daemon-reload
sudo systemctl enable telegram-claude-bot
sudo systemctl start telegram-claude-bot

# Verify it's running
sudo systemctl status telegram-claude-bot

# View logs
sudo journalctl -u telegram-claude-bot -f
```

**Common commands:**
```bash
sudo systemctl start telegram-claude-bot      # Start
sudo systemctl stop telegram-claude-bot       # Stop
sudo systemctl restart telegram-claude-bot    # Restart
sudo systemctl enable telegram-claude-bot     # Auto-start on boot
sudo journalctl -u telegram-claude-bot -n 100 # Last 100 log lines
```

---

### Scenario 4: Docker Container

The project includes a production-ready `Dockerfile` (multi-stage: Go builder + `node:22-alpine` runtime with Claude CLI) and `docker-compose.yml`.

**Step 1: Configure**

```bash
cp .env.docker.example .env.docker
nano .env.docker
# Required: TELEGRAM_BOT_TOKEN, ADMIN_TELEGRAM_IDS, ANTHROPIC_API_KEY
```

**Step 2: Build and run**

```bash
make docker-build    # Build Docker image
make docker-up       # Start container (detached)
make docker-logs     # Follow logs
```

**Step 3: Verify**

```bash
curl http://localhost:3000/health
# {"status":"ok","uptime_seconds":5}
```

**Docker architecture:**
- **Builder stage:** `golang:1.25-alpine` — compiles static Go binary (`CGO_ENABLED=0`)
- **Runtime stage:** `node:22-alpine` — Claude CLI installed globally via npm
- **Non-root user:** `appuser` with `HOME=/app`
- **Entrypoint:** `scripts/docker-entrypoint.sh` — validates data dir permissions before starting
- **Volume:** `bot-data:/app/data` — persists all JSON/Markdown data across restarts
- **Resource limits:** 4GB RAM, 2 CPUs
- **Log rotation:** `json-file` driver, 10MB max, 3 files
- **Healthcheck:** `wget` to `/health` every 30s with 10s start period

**Make targets:**

| Target | Description |
|--------|-------------|
| `make docker-build` | Build image via `docker compose build` |
| `make docker-build-amd64` | Cross-build for linux/amd64 (`docker buildx`) |
| `make docker-up` | Start (auto-creates `.env.docker` from example if missing) |
| `make docker-down` | Stop containers |
| `make docker-logs` | Follow container logs |

**Cross-platform deployment (e.g., ARM Mac → x86 VPS):**

```bash
make docker-build-amd64
# Then push/transfer the image to your server
```

---

### Scenario 5: Kubernetes Deployment

**Step 1: Create Docker image and push to registry**

```bash
docker build -t myregistry/telegram-claude-bot:v1.0 .
docker push myregistry/telegram-claude-bot:v1.0
```

**Step 2: Create Kubernetes manifests**

`k8s/deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: telegram-claude-bot
  labels:
    app: telegram-claude-bot
spec:
  replicas: 1
  selector:
    matchLabels:
      app: telegram-claude-bot
  template:
    metadata:
      labels:
        app: telegram-claude-bot
    spec:
      containers:
      - name: bot
        image: myregistry/telegram-claude-bot:v1.0
        ports:
        - containerPort: 3000
        env:
        - name: TELEGRAM_BOT_TOKEN
          valueFrom:
            secretKeyRef:
              name: telegram-secrets
              key: bot-token
        - name: ADMIN_TELEGRAM_IDS
          valueFrom:
            configMapKeyRef:
              name: telegram-config
              key: admin-ids
        volumeMounts:
        - name: data
          mountPath: /app/data
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: telegram-bot-pvc
```

`k8s/service.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: telegram-claude-bot-service
spec:
  selector:
    app: telegram-claude-bot
  ports:
  - protocol: TCP
    port: 80
    targetPort: 3000
  type: LoadBalancer
```

`k8s/pvc.yaml`:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: telegram-bot-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
```

**Step 3: Deploy**

```bash
# Create secrets
kubectl create secret generic telegram-secrets \
  --from-literal=bot-token=your_token

# Create configmap
kubectl create configmap telegram-config \
  --from-literal=admin-ids=123456789

# Deploy
kubectl apply -f k8s/

# Verify
kubectl get pods
kubectl logs -f deployment/telegram-claude-bot
```

---

## Database Migration (Future)

Currently uses file-based storage. Future versions will support SQL database.

**Migration path (v2.0+):**
1. Export data from `data/` directory
2. Run migration script
3. Start bot pointing to database
4. Archive old `data/` directory

---

## Monitoring & Health Checks

### Health Endpoint

```bash
curl http://localhost:3000/health

# Response:
# {
#   "status": "ok",
#   "uptime_seconds": 3600,
#   "version": "1.0.0",
#   "timestamp": "2025-03-16T14:23:45Z"
# }
```

### Log Monitoring

**Development:**
```bash
go run ./cmd/bot/ 2>&1 | grep -i error
```

**Systemd:**
```bash
sudo journalctl -u telegram-claude-bot -f | grep ERROR
```

**Docker:**
```bash
docker logs -f telegram-claude-bot | grep ERROR
```

### Dashboard Access

Navigate to `http://localhost:3000` (or your server IP):

- `/health` — Health check
- `/api/users` — List users (requires `X-API-Key` header)
- `/api/stats` — System statistics
- `/api/config` — Configuration
- `/ws` — WebSocket real-time events

### Metrics to Monitor

| Metric | Alert Threshold |
|--------|-----------------|
| Disk usage (data/) | >80% of available |
| Process CPU | >80% sustained |
| Process memory | >500MB |
| Goroutine count | >1000 (leak indicator) |
| Query error rate | >1% |
| Average query latency | >5 seconds (p95) |

---

## Troubleshooting

### Issue: "Bot started but not responding to messages"

**Cause:** Whitelist not set up or token incorrect.

**Solution:**
```bash
# Check token works
curl https://api.telegram.org/bot{TOKEN}/getMe

# Check admin ID is set
cat .env | grep ADMIN_TELEGRAM_IDS

# Try adding yourself to whitelist
# Use /admin whitelist {your_id} from another admin account
```

### Issue: "Claude CLI not found"

**Cause:** Claude CLI not installed or not in PATH.

**Solution:**
```bash
# Install Claude CLI
curl -L https://cdn.anthropic.com/cli/install.sh | bash

# Verify it's in PATH
which claude

# Test it works
claude --version
```

### Issue: "File permission denied"

**Cause:** `data/` directory not writable by bot process.

**Solution:**
```bash
# Check permissions
ls -la data/

# Fix ownership (systemd case)
sudo chown -R botuser:botuser /opt/telegram-claude-bot/data

# Fix permissions
sudo chmod -R 755 /opt/telegram-claude-bot/data
```

### Issue: "High CPU usage"

**Cause:** Usually from rate limiter cleanup or session compaction running.

**Solution:**
```bash
# Check goroutine count
curl localhost:3000/health | jq .goroutines

# If >1000, likely a leak
# File an issue with `go pprof` output:
curl localhost:3000/debug/pprof/goroutine > goroutines.txt
```

### Issue: "Port 3000 already in use"

**Cause:** Another process listening on port 3000.

**Solution:**
```bash
# Find process
lsof -i :3000

# Kill it (if not needed)
kill -9 PID

# Or change WEB_PORT in .env
WEB_PORT=3001
```

### Issue: "Out of memory"

**Cause:** Old sessions not being compacted, or large session files.

**Solution:**
```bash
# Check largest files
du -sh data/sessions/*/* | sort -h | tail -10

# Manually trigger compaction
# (In future version, via API)

# Or restart bot to free memory
systemctl restart telegram-claude-bot
```

---

## Backup & Recovery

### Automated Daily Backup

Create `/opt/telegram-claude-bot/backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/mnt/backups/telegram-bot"
SOURCE_DIR="/opt/telegram-claude-bot/data"

mkdir -p "$BACKUP_DIR"

# Create timestamped backup
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
tar -czf "$BACKUP_DIR/backup_$TIMESTAMP.tar.gz" -C "$SOURCE_DIR" .

# Keep only last 30 days
find "$BACKUP_DIR" -name "backup_*.tar.gz" -mtime +30 -delete

echo "Backup completed: $BACKUP_DIR/backup_$TIMESTAMP.tar.gz"
```

**Schedule via cron:**
```bash
# Daily at 2 AM
0 2 * * * /opt/telegram-claude-bot/backup.sh >> /var/log/telegram-bot-backup.log 2>&1
```

### Restore from Backup

```bash
# Stop bot
sudo systemctl stop telegram-claude-bot

# List backups
ls -la /mnt/backups/telegram-bot/

# Extract backup
cd /opt/telegram-claude-bot
tar -xzf /mnt/backups/telegram-bot/backup_20250316_020000.tar.gz

# Restart bot
sudo systemctl start telegram-claude-bot
```

---

## Security Best Practices

### 1. Restrict File System Access

```bash
# Only allow bot to access specific directories
ALLOWED_WORKING_DIRS=/home/user/projects,/tmp

# Users cannot access other directories via /project command
```

### 2. Protect Secrets

```bash
# Never commit .env to git
echo ".env" >> .gitignore

# File permissions on .env
chmod 600 .env

# In systemd service
chmod 600 /opt/telegram-claude-bot/.env
```

### 3. API Key Security

```bash
# Use strong random key
ADMIN_API_KEY=$(openssl rand -hex 32)

# Rotate regularly
# (Update in .env and data/config.json)

# Monitor API access (logs include API key usage)
```

### 4. Update Dependencies

```bash
# Check for vulnerabilities
go list -m all | go mod verify

# Update to latest
go get -u ./...

# Rebuild and deploy
go build -o dist/telegram-claude-bot ./cmd/bot/
```

### 5. Firewall Rules

```bash
# Only expose necessary ports
# 3000: Dashboard (admin access only)
# Telegram API: Outbound only

# Linux UFW example
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 3000/tcp  # Dashboard (restrict source IPs)
sudo ufw enable
```

---

## Performance Tuning

### Increase Concurrency (if hardware allows)

```bash
MAX_CONCURRENT_CLI_PROCESSES=5  # Default: 3
RATE_LIMIT_REQUESTS_PER_MINUTE=20  # Default: 10
```

### Adjust Context Window

```bash
CLAUDE_CONTEXT_MESSAGES=15  # More context, slower queries
CLAUDE_CONTEXT_MESSAGES=5   # Less context, faster queries
```

### Session Compaction

```bash
COMPACT_THRESHOLD=30         # Compact after 30 messages
COMPACT_KEEP_RECENT=8        # Keep 8 recent messages
COMPACT_ENABLED=true         # Run in background
```

### Fast SSD for data/ Directory

```bash
# Move data/ to SSD
sudo mv /opt/telegram-claude-bot/data /var/local/telegram-bot-data
sudo ln -s /var/local/telegram-bot-data /opt/telegram-claude-bot/data
```

---

## Updating to New Versions

### Before Update

```bash
# Backup current data
tar -czf telegram-bot-backup-$(date +%Y%m%d).tar.gz data/

# Record current version
git log -1 --format="%H %s" > .version-backup
```

### Update Steps

```bash
# Stop bot
sudo systemctl stop telegram-claude-bot

# Pull latest code
git pull origin main

# Review changes (if any breaking changes in CHANGELOG.md)
cat CHANGELOG.md

# Build new binary
go mod tidy
go build -o dist/telegram-claude-bot ./cmd/bot/

# Copy to deployment directory
sudo cp dist/telegram-claude-bot /opt/telegram-claude-bot/

# Start bot
sudo systemctl start telegram-claude-bot

# Verify health
curl http://localhost:3000/health
```

### Rollback (if issues)

```bash
# Stop bot
sudo systemctl stop telegram-claude-bot

# Restore data
tar -xzf telegram-bot-backup-20250316.tar.gz

# Restore previous binary
git checkout previous-version
go build -o dist/telegram-claude-bot ./cmd/bot/
sudo cp dist/telegram-claude-bot /opt/telegram-claude-bot/

# Start bot
sudo systemctl start telegram-claude-bot
```

---

## Support & Troubleshooting

For detailed troubleshooting, see:
- [System Architecture](./system-architecture.md) — Internal design
- [Code Standards](./code-standards.md) — Development guidelines
- [README.md](../README.md) — User documentation

For production support, contact the development team or file an issue on GitHub.

