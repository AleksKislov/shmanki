# Deploying Shmanki on DigitalOcean

## Droplet Requirements (test/approbation)

| Resource  | Value                     |
| --------- | ------------------------- |
| Plan      | Basic (Regular, shared)   |
| vCPU      | 1                         |
| RAM       | 1 GB                      |
| Disk      | 25 GB SSD                 |
| OS        | Ubuntu 24.04 LTS          |
| Public IP | Included                  |

Estimated cost: **$6/month** (or free for ~2 months if you use DigitalOcean's new-account signup credit).

1 GB of RAM is tight for four containers (Postgres, Go backend, Node SSR frontend, nginx) running side by side, especially during `docker compose build`. Step 4 below adds a swap file to cover the gap — don't skip it. If you'd rather have headroom and not think about it, size up to the $12/mo 2 GB droplet instead and skip the swap step.

---

## 1. Create the Droplet

### Option A: DigitalOcean Console

1. Go to **Droplets → Create Droplet**
2. Choose a region close to you
3. Image: **Ubuntu 24.04 LTS**
4. Size: **Basic → Regular → $6/mo (1 GB / 1 vCPU / 25 GB SSD)**
5. Authentication: add your SSH key
6. Create Droplet

### Option B: `doctl` CLI

```bash
# Install doctl if you haven't
brew install doctl   # macOS
# or: snap install doctl

# Authenticate (creates an API token in the DO console)
doctl auth init

# Find your SSH key fingerprint
doctl compute ssh-key list

# Create the droplet
doctl compute droplet create shmanki-test \
  --region nyc1 \
  --image ubuntu-24-04-x64 \
  --size s-1vcpu-1gb \
  --ssh-keys <YOUR_SSH_KEY_FINGERPRINT>
```

Get the public IP:

```bash
doctl compute droplet list
```

---

## 2. Open Firewall Ports

### DigitalOcean Cloud Firewall

In the console: **Networking → Firewalls → Create Firewall**

Add inbound rules for:

- TCP port 22 (SSH)
- TCP port 80 (HTTP)

Apply it to your droplet.

Or via CLI:

```bash
doctl compute firewall create \
  --name shmanki-fw \
  --inbound-rules "protocol:tcp,ports:22,address:0.0.0.0/0,address:::/0 protocol:tcp,ports:80,address:0.0.0.0/0,address:::/0" \
  --droplet-ids <DROPLET_ID>
```

### VM-level firewall (ufw)

SSH into the droplet first, then:

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw enable
```

**Important:** Add both rules before running `ufw enable` — otherwise you may lose your SSH session.

---

## 3. SSH Into the Droplet

```bash
ssh -i ~/.ssh/<YOUR_KEY> root@<DROPLET_IP>
```

DigitalOcean's Ubuntu images log you in as `root` by default.

---

## 4. Add Swap (Recommended for 1 GB Droplets)

A 1 GB droplet has no swap by default. Without it, `docker compose build` or a Postgres+backend+frontend burst can trigger the OOM killer. Add a 2 GB swapfile (2× RAM is the standard rule of thumb below 2 GB):

```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Persist across reboots
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

# Prefer RAM over swap when both are available (swap is just a safety net here)
echo 'vm.swappiness=10' | sudo tee -a /etc/sysctl.conf
sudo sysctl vm.swappiness=10

# Verify
free -h
```

If you sized up to the 2 GB droplet instead, you can skip this step.

---

## 5. Install Docker

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
  sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Add Docker repo
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Verify
docker --version
docker compose version
```

You're already `root`, so there's no need for `usermod -aG docker` / `newgrp docker` here — skip that if you're following along from memory of other guides.

---

## 6. Clone the Repository

```bash
cd ~
git clone https://github.com/<YOUR_USER>/shmanki.git
cd shmanki
```

---

## 7. Configure Environment

Create `.env.prod` in the project root. **This file is never committed — create it manually on the droplet.**

```bash
cat > .env.prod << 'EOF'
# Database
POSTGRES_DB=shmanki
POSTGRES_USER=postgres
POSTGRES_PASSWORD=hello_there_shmanki_456!

# Backend
PORT=8080
ENV=production
DATABASE_URL=postgres://postgres:hello_there_shmanki_456!@db:5432/shmanki?sslmode=disable
JWT_SECRET=REPLACE_WITH_64_CHAR_RANDOM_STRING
JWT_TTL_HOURS=168

# LLM
LLM_API_URL=https://api.openai.com/v1/chat/completions
LLM_API_KEY=REPLACE_WITH_YOUR_KEY
LLM_MODEL=gpt-4.1-mini
LLM_PROVIDER=openai-compatible
LLM_TIMEOUT_SECONDS=30

# App
DEFAULT_LANGUAGE=en
EOF
```

Generate secure values for the secrets:

```bash
# DB password
openssl rand -base64 16

# JWT secret
openssl rand -hex 32
```

Replace the placeholder values in `.env.prod` with the generated output:

```bash
nano .env.prod
```

**Make sure `POSTGRES_PASSWORD` in `DATABASE_URL` matches the `POSTGRES_PASSWORD` value exactly.**

---

## 8. First Deploy

```bash
cd ~/shmanki

# Build and start all services (db, backend, frontend, nginx)
docker compose -f compose.prod.yaml --env-file .env.prod up -d --build

# Run database migrations
docker compose -f compose.prod.yaml --env-file .env.prod --profile tools run --rm migrate

# Check everything is running
docker compose -f compose.prod.yaml ps
```

The app is now available at `http://<DROPLET_IP>`.

Check logs if something is wrong:

```bash
docker compose -f compose.prod.yaml logs -f
```

---

## 9. Redeployment

After pushing new code to GitHub:

```bash
cd ~/shmanki
git pull
docker compose -f compose.prod.yaml --env-file .env.prod up -d --build
```

If there are new migrations:

```bash
docker compose -f compose.prod.yaml --profile tools run --rm migrate
```

Remove old unused images to free disk space:

```bash
docker image prune -f
```

### One-command redeploy script

Save this to `~/redeploy.sh` on the droplet (run once):

```bash
cat > ~/redeploy.sh << 'EOF'
#!/bin/bash
set -e
cd ~/shmanki
git pull origin main
docker compose -f compose.prod.yaml --env-file .env.prod up -d --build
docker compose -f compose.prod.yaml --env-file .env.prod --profile tools run --rm migrate
docker image prune -f
echo "Deploy complete!"
EOF
chmod +x ~/redeploy.sh
```

Then every future deploy is just:

```bash
bash ~/redeploy.sh
```

---

## 10. Useful Commands

```bash
# View live logs
docker compose -f compose.prod.yaml logs -f
docker compose -f compose.prod.yaml logs -f backend
docker compose -f compose.prod.yaml logs -f frontend

# Restart a single service
docker compose -f compose.prod.yaml restart backend

# Open a DB shell
docker compose -f compose.prod.yaml exec db psql -U postgres -d shmanki

# Check memory / swap usage
free -h

# Stop everything (keeps data)
docker compose -f compose.prod.yaml down

# Stop and wipe DB volume — destroys all data
docker compose -f compose.prod.yaml down -v
```

---

## 11. Project Files Relevant to Deployment

These files are in the repo and used during deployment:

```
shmanki/
├── backend/
│   └── Dockerfile              # Go multi-stage build
├── frontend/
│   ├── Dockerfile              # Node.js SSR build
│   └── adapters/
│       └── node-server/
│           └── vite.config.ts  # Qwik Node.js adapter
├── deploy/
│   └── nginx.conf              # Nginx reverse proxy config
├── compose.prod.yaml           # Production Docker Compose
└── compose.dev.yaml            # Local dev Docker Compose
```

---

## Troubleshooting

| Issue                             | Fix                                                                                            |
| ---------------------------------- | ---------------------------------------------------------------------------------------------- |
| Can't SSH after `ufw enable`       | Use the DigitalOcean web console to access the droplet: **Droplet → Access → Launch Droplet Console** |
| Port 80 unreachable                | Check the DigitalOcean Cloud Firewall has an inbound TCP 80 rule and is applied to the droplet |
| Build/deploy killed unexpectedly   | Likely OOM on the 1 GB droplet — confirm swap is active with `free -h`, or resize to 2 GB      |
| Backend crashes on start           | Check `.env.prod` exists and `DATABASE_URL` uses `db` as hostname, not `localhost`             |
| `migrate` fails                    | Ensure db container is healthy: `docker compose -f compose.prod.yaml ps`                       |
| `build.server` fails in frontend   | Run `npm install` locally, commit updated `package-lock.json`                                  |
| DB password mismatch               | `POSTGRES_PASSWORD` in `.env.prod` must be identical in both the db section and `DATABASE_URL` |
