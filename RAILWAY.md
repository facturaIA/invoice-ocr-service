# Railway Deployment Guide

Complete guide for deploying the Invoice OCR Service to Railway with memory optimization for the free tier (< 512MB RAM).

## Quick Start

### 1. Prerequisites

- Railway account: https://railway.app/
- Railway CLI installed: `npm install -g @railway/cli`
- Git repository (GitHub, GitLab, or Bitbucket)

### 2. Deployment Steps

```bash
# Login to Railway
railway login

# Initialize project (in this directory)
railway init

# Set required environment variable
railway variables set GEMINI_API_KEY=your-api-key-here

# Deploy
railway up
```

### 3. Verify Deployment

```bash
# View logs
railway logs

# Open dashboard
railway open

# Check health endpoint
curl https://your-service.railway.app/health
```

---

## Memory Optimization

This service is optimized to run within Railway's free tier limit of **512MB RAM**.

### Configuration Applied

#### 1. Dockerfile Optimizations

**Build-time optimizations:**
```dockerfile
# Strip debug symbols and reduce binary size
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-s -w" \
    -o server \
    ./cmd/server
```
- `-ldflags="-s -w"` removes debug info and symbol table (~30% size reduction)

**Runtime optimizations:**
```dockerfile
# Go garbage collection tuning
ENV GOGC=50
ENV GOMEMLIMIT=450MiB
```
- `GOGC=50`: More aggressive garbage collection (default: 100)
- `GOMEMLIMIT=450MiB`: Hard memory limit for Go runtime

#### 2. railway.toml Configuration

```toml
[deploy.memory]
limit = 512      # Maximum memory in MB
request = 256    # Requested memory in MB
```

#### 3. Application-Level Optimizations

**File upload limits:**
- Max file size: 10MB (configurable via `MAX_FILE_SIZE`)
- Prevents memory exhaustion from large uploads

**Processing timeouts:**
- Default timeout: 30 seconds (configurable via `PROCESSING_TIMEOUT`)
- Prevents long-running operations from accumulating

**Stateless design:**
- No in-memory caching
- No session storage
- Each request is independent

---

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `GEMINI_API_KEY` | Google Gemini API key | `AIza...` |

Set in Railway dashboard or CLI:
```bash
railway variables set GEMINI_API_KEY=your-key
```

### Optional (with defaults)

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port (auto-set by Railway) |
| `HOST` | `0.0.0.0` | Server host |
| `TESSERACT_LANG` | `spa+eng` | OCR language codes |
| `MAX_FILE_SIZE` | `10485760` | Max upload size (bytes) |
| `PROCESSING_TIMEOUT` | `30` | Processing timeout (seconds) |
| `AI_DEFAULT_PROVIDER` | `gemini` | AI provider (gemini/openai/ollama) |
| `GEMINI_MODEL` | `gemini-pro` | Gemini model name |
| `ENABLE_VISION_MODEL` | `false` | Use vision models by default |
| `ENABLE_DEBUG_MODE` | `false` | Enable debug logging |

### Auto-Set by Railway

| Variable | Description |
|----------|-------------|
| `RAILWAY_ENVIRONMENT` | Environment name (production/staging) |
| `RAILWAY_SERVICE_NAME` | Service name |

---

## Monitoring

### Health Check Endpoint

Railway monitors the service via the `/health` endpoint configured in `railway.toml`:

```toml
[healthcheck]
path = "/health"
interval = 30
timeout = 10
```

### Health Response Example

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2024-01-15T14:30:00Z",
  "uptime": "2h15m30s",
  "memory": {
    "allocated": "125.50 MB",
    "total": "450.00 MB",
    "system": "180.00 MB"
  },
  "tesseract": {
    "available": true,
    "version": "tesseract 5.3.0"
  },
  "imageMagick": {
    "available": true,
    "version": "Version: ImageMagick 7.1.1"
  },
  "ai": {
    "defaultProvider": "gemini",
    "ocrEngine": "tesseract"
  }
}
```

### Status Codes

- **200 OK**: Service is healthy and all dependencies are available
- **503 Service Unavailable**: Service is degraded (Tesseract or ImageMagick unavailable)

---

## Memory Usage Patterns

### Expected Memory Usage

| Scenario | Memory Usage |
|----------|-------------|
| **Idle** | 50-80 MB |
| **Processing 1MB image** | 150-250 MB |
| **Processing 5MB image** | 250-350 MB |
| **Processing 10MB image** | 350-450 MB |

### Memory Monitoring

Check current memory usage via the health endpoint:

```bash
curl https://your-service.railway.app/health | jq '.memory'
```

### Railway Dashboard

1. Open Railway dashboard: `railway open`
2. Navigate to "Metrics" tab
3. Monitor:
   - Memory usage over time
   - CPU usage
   - Network traffic
   - Request count

---

## Troubleshooting

### Issue: Out of Memory (OOM) Errors

**Symptoms:**
- Service restarts frequently
- 503 errors under load
- Railway dashboard shows memory spikes to 512MB

**Solutions:**

1. **Reduce max file size:**
```bash
railway variables set MAX_FILE_SIZE=5242880  # 5MB instead of 10MB
```

2. **Increase GOGC (less aggressive GC):**
Add to Dockerfile:
```dockerfile
ENV GOGC=75  # Default: 50
```

3. **Use vision models (skip OCR):**
Vision models process images more efficiently:
```bash
railway variables set ENABLE_VISION_MODEL=true
```

4. **Upgrade Railway plan:**
Free tier: 512MB → Hobby tier: 8GB

### Issue: Slow Processing

**Symptoms:**
- Requests timeout (> 30s)
- High latency on API calls

**Solutions:**

1. **Increase processing timeout:**
```bash
railway variables set PROCESSING_TIMEOUT=60
```

2. **Use faster AI model:**
```bash
# Gemini Pro is faster than GPT-4
railway variables set AI_DEFAULT_PROVIDER=gemini
railway variables set GEMINI_MODEL=gemini-pro
```

3. **Check health endpoint:**
```bash
curl https://your-service.railway.app/health
```
Verify Tesseract and ImageMagick are available.

### Issue: Health Check Failures

**Symptoms:**
- Railway shows service as "unhealthy"
- Automatic restarts

**Solutions:**

1. **Check logs:**
```bash
railway logs --tail=100
```

2. **Test health endpoint locally:**
```bash
docker run -p 8080:8080 -e GEMINI_API_KEY=test invoice-ocr-service
curl http://localhost:8080/health
```

3. **Verify dependencies:**
The health endpoint checks Tesseract and ImageMagick. If either is missing, the service reports as degraded.

### Issue: API Key Errors

**Symptoms:**
- "API key invalid" errors
- AI extraction failures

**Solutions:**

1. **Verify API key is set:**
```bash
railway variables
```

2. **Re-set API key:**
```bash
railway variables set GEMINI_API_KEY=your-new-key
```

3. **Check API key validity:**
Test the key directly:
```bash
curl "https://generativelanguage.googleapis.com/v1/models?key=YOUR_KEY"
```

---

## Cost Optimization

### Railway Costs

**Free Tier:**
- $5 free credit per month
- Usage-based pricing after that
- This service typically uses ~$2-5/month on free tier

**Usage breakdown:**
- Memory: 512MB × 24/7 ≈ $1-2/month
- CPU: Low usage ≈ $0.50/month
- Network: ~10GB/month ≈ $0.50/month

### AI API Costs

**Google Gemini (Recommended):**
- Free tier: 60 requests/minute
- Paid: $0.00025 per 1K characters
- ~1,000 invoices/month = ~$0.50

**OpenAI GPT-4:**
- $0.03 per 1K tokens (input)
- ~1,000 invoices/month = ~$30-50
- Much more expensive than Gemini

### Cost-Saving Tips

1. **Use Gemini instead of OpenAI** (10-100x cheaper)
2. **Set reasonable file size limits** (reduce memory/processing)
3. **Use vision models for complex receipts only** (OCR path is cheaper)
4. **Monitor usage in Railway dashboard**

---

## Scaling

### Horizontal Scaling

Railway automatically handles horizontal scaling:

1. **Enable autoscaling** in Railway dashboard
2. **Set replica count** (2-5 replicas for production)
3. **Load balancing** is automatic

### Vertical Scaling

If you need more resources:

1. **Upgrade Railway plan:**
   - Free: 512MB RAM
   - Hobby: 8GB RAM
   - Pro: 32GB RAM

2. **Update railway.toml:**
```toml
[deploy.memory]
limit = 2048      # 2GB
request = 1024
```

### Performance Tips

1. **Cache Gemini responses** (if processing same invoices repeatedly)
2. **Use CDN** for static assets
3. **Implement request queuing** for burst traffic
4. **Add rate limiting** to prevent abuse

---

## CI/CD Integration

### GitHub Actions

Create `.github/workflows/railway.yml`:

```yaml
name: Deploy to Railway

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Railway CLI
        run: npm install -g @railway/cli

      - name: Deploy to Railway
        run: railway up --detach
        env:
          RAILWAY_TOKEN: ${{ secrets.RAILWAY_TOKEN }}
```

**Setup:**
1. Get Railway token: `railway login` → Get token from dashboard
2. Add to GitHub secrets: Settings → Secrets → `RAILWAY_TOKEN`
3. Push to main branch to trigger deployment

### Manual Deployment

```bash
# Deploy from local machine
railway up

# Deploy specific branch
git checkout production
railway up

# Rollback to previous deployment
railway rollback
```

---

## Security Best Practices

### 1. API Key Management

**❌ Don't:**
- Commit API keys to git
- Share API keys in public channels
- Use the same key for dev/prod

**✅ Do:**
- Use Railway's environment variables
- Rotate keys regularly
- Use different keys per environment

### 2. Rate Limiting

Add to your reverse proxy (nginx/Caddy):

```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;

server {
    location /api/process-invoice {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://railway-service;
    }
}
```

### 3. File Upload Validation

The service already validates:
- File size (max 10MB)
- File type (images only)
- Processing timeout (30s)

### 4. HTTPS

Railway provides automatic HTTPS for all deployments. No configuration needed!

---

## Support

### Documentation
- Railway Docs: https://docs.railway.app/
- Invoice OCR Service: See main README.md

### Debugging

Enable debug mode:
```bash
railway variables set ENABLE_DEBUG_MODE=true
railway logs --tail=100
```

### Common Logs

**Successful processing:**
```
[INFO] Image preprocessing completed in 0.5s
[INFO] OCR extraction completed in 1.2s
[INFO] AI extraction completed in 2.3s
[INFO] Total processing time: 4.0s
```

**Error:**
```
[ERROR] Image preprocessing failed: tesseract not found
[ERROR] AI extraction failed: invalid API key
```

### Getting Help

1. Check logs: `railway logs`
2. Check health: `curl /health`
3. Test locally: `docker build && docker run`
4. File issue: https://github.com/facturaIA/invoice-ocr-service/issues

---

## Example: Complete Deployment

```bash
# 1. Clone repository
git clone https://github.com/facturaIA/invoice-ocr-service
cd invoice-ocr-service

# 2. Login to Railway
railway login

# 3. Create new project
railway init

# 4. Set environment variables
railway variables set GEMINI_API_KEY=AIza...
railway variables set TESSERACT_LANG=spa+eng
railway variables set AI_DEFAULT_PROVIDER=gemini

# 5. Deploy
railway up

# 6. Get service URL
railway status

# 7. Test deployment
curl https://your-service.railway.app/health

# 8. Process an invoice
curl -X POST https://your-service.railway.app/api/process-invoice \
  -F "file=@invoice.jpg" \
  -F "aiProvider=gemini"

# 9. Monitor
railway logs --tail=50
railway open  # Opens dashboard
```

---

## Migration from Other Platforms

### From Heroku

Similar to Heroku dynos:
- Railway = Heroku dyno
- `railway.toml` = `Procfile`
- Railway env vars = Heroku config vars

Key differences:
- Railway has better free tier (512MB vs Heroku's 512MB)
- Railway auto-deploys from git
- Railway provides automatic HTTPS

### From Docker Compose

Railway uses the same Dockerfile:
- Keep your `Dockerfile`
- Remove `docker-compose.yml` (Railway handles orchestration)
- Set env vars in Railway dashboard instead of `.env` files

### From VPS (Digital Ocean, AWS EC2)

Railway simplifies deployment:
- No need to manage nginx/Caddy (Railway provides ingress)
- No need to manage SSL certificates (automatic HTTPS)
- No need to manage systemd services (Railway handles restarts)
- No need to manage firewall rules (Railway handles networking)

---

## Next Steps

1. **Deploy to Railway** using the quick start above
2. **Test the API** with sample invoices
3. **Monitor performance** in Railway dashboard
4. **Set up CI/CD** with GitHub Actions
5. **Configure alerts** for downtime/errors
6. **Scale as needed** based on usage patterns

For production deployments, consider:
- Upgrading to Railway Hobby plan ($5/month) for higher limits
- Setting up staging environment (separate Railway project)
- Implementing comprehensive logging (Papertrail, Logtail)
- Adding application monitoring (Sentry, New Relic)
- Creating backups of processed data
