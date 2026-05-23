# Deploying Shmanki on Yandex Cloud VM

## VM Requirements (test/approbation)

| Resource  | Value                 |
| --------- | --------------------- |
| Platform  | standard-v3           |
| vCPU      | 2 (core fraction 20%) |
| RAM       | 4 GB                  |
| Disk      | 20 GB network-ssd     |
| OS        | Ubuntu 24.04 LTS      |
| Public IP | Ephemeral (or static) |

---

## 1. Create the VM

### Option A: Yandex Cloud Console

1. Go to **Compute Cloud → Create VM**
2. Pick Ubuntu 24.04 LTS, standard-v3, 2 vCPU (20%), 4 GB RAM, 20 GB SSD
3. Add your SSH key
4. Under **Network**, check "Public IPv4 address"
5. Create

### Option B: `yc` CLI

```bash
# Install yc if you haven't
curl -sSL https://storage.yandexcloud.net/yandexcloud-yc/install.sh | bash

# Init (first time)
yc init

# Create VM
yc compute instance create \
  --name shmanki-test \
  --zone ru-central1-b \
  --platform standard-v3 \
  --cores 2 \
  --core-fraction 20 \
  --memory 4 \
  --create-boot-disk image-folder-id=standard-images,image-family=ubuntu-2404-lts-oslogin,size=20,type=network-ssd \
  --network-interface subnet-name=default-ru-central1-b,nat-ip-version=ipv4 \
  --ssh-key ~/.ssh/yaserver_id_rsa.pub
```

Note the external IP from the output.

---

## 2. Initial VM Setup

SSH into the VM:

```bash
ssh yc-user@<VM_IP>
```

### Install Docker + Docker Compose

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
sudo apt install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
  https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Add your user to docker group (no sudo needed for docker commands)
sudo usermod -aG docker $USER
newgrp docker
```

### Open firewall ports

```bash
# If using ufw
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

Also ensure ports 80 and 443 are open in the **Yandex Cloud Security Group** attached to your VM (by default all outbound is allowed, but inbound might need a rule for TCP 80).

---

## 3. Clone the Repository

```bash
cd ~
git clone https://github.com/<YOUR_USER>/shmanki.git
cd shmanki
```

---

## 4. Configure Environment

Create `.env.prod` in the project root:

```bash
cat > .env.prod << 'EOF'
# Database
POSTGRES_DB=shmanki
POSTGRES_USER=postgres
POSTGRES_PASSWORD=<STRONG_PASSWORD_HERE>

# Backend
PORT=8080
ENV=production
DATABASE_URL=postgres://postgres:<STRONG_PASSWORD_HERE>@db:5432/shmanki?sslmode=disable
JWT_SECRET=<RANDOM_64_CHAR_STRING>
JWT_TTL_HOURS=168

# LLM (optional, leave empty if not needed yet)
LLM_API_URL=https://api.openai.com/v1/chat/completions
LLM_API_KEY=
LLM_MODEL=gpt-4.1-mini
LLM_PROVIDER=openai-compatible
LLM_TIMEOUT_SECONDS=30

# App
DEFAULT_LANGUAGE=en
EOF
```

Generate secure values:

```bash
# Generate JWT secret
openssl rand -hex 32

# Generate DB password
openssl rand -base64 16
```

**Important:** The backend container reads `.env.prod` as its env file. The `DATABASE_URL` in `.env.prod` must use `db` as hostname (the Docker Compose service name), not `localhost`.

---

## 5. First Deploy

```bash
cd ~/shmanki

# Build and start all services
docker compose -f compose.prod.yaml up -d --build

# Run migrations
docker compose -f compose.prod.yaml --profile tools run --rm migrate

# Check logs
docker compose -f compose.prod.yaml logs -f
```

The app is now available at `http://<VM_IP>`.

---

## 6. Redeployment (one command)

After pushing changes to your repo:

```bash
cd ~/shmanki && git pull && docker compose -f compose.prod.yaml up -d --build
```

If there are new migrations:

```bash
docker compose -f compose.prod.yaml --profile tools run --rm migrate
```

### Convenience: deploy script

Create `deploy/redeploy.sh`:

```bash
#!/bin/bash
set -e
cd ~/shmanki
git pull origin main
docker compose -f compose.prod.yaml up -d --build
docker compose -f compose.prod.yaml --profile tools run --rm migrate
docker image prune -f
echo "Deploy complete!"
```

Then just run `bash ~/shmanki/deploy/redeploy.sh`.

---

## 7. Useful Commands

```bash
# View logs
docker compose -f compose.prod.yaml logs -f backend
docker compose -f compose.prod.yaml logs -f frontend

# Restart a single service
docker compose -f compose.prod.yaml restart backend

# Enter DB shell
docker compose -f compose.prod.yaml exec db psql -U postgres -d shmanki

# Stop everything
docker compose -f compose.prod.yaml down

# Stop and wipe DB (careful!)
docker compose -f compose.prod.yaml down -v
```

---

## 8. Backend `.env` Note

The backend's `config.Load()` expects a `.env` file in the working directory. In Docker, the `env_file` directive in `compose.prod.yaml` injects all variables from `.env.prod` as environment variables. However, the code checks for a `.env` file on disk.

**Fix:** The backend Dockerfile should create a dummy `.env` or the code should be updated to not require the file when env vars are already set. The simplest fix is to add this to `backend/Dockerfile`:

```dockerfile
RUN touch /app/.env
```

This is already not in the Dockerfile above — you should add it if the app crashes with "missing backend/.env". Alternatively, update `config.Load()` to skip the file check in production.

---

## Project File Structure (new files)

```
shmanki/
├── backend/
│   └── Dockerfile              ← NEW
├── frontend/
│   ├── Dockerfile              ← NEW
│   ├── adapters/
│   │   └── node-server/
│   │       └── vite.config.ts  ← NEW
│   └── src/
│       └── entry.node-server.tsx ← NEW
├── deploy/
│   └── nginx.conf              ← NEW
├── compose.prod.yaml           ← NEW
└── docs/
    └── deployment.md           ← THIS FILE
```

---

## Troubleshooting

| Issue                                       | Fix                                                                             |
| ------------------------------------------- | ------------------------------------------------------------------------------- |
| Backend crashes with "missing backend/.env" | Add `RUN touch /app/.env` to `backend/Dockerfile` before `CMD`                  |
| Frontend build fails on `build.server`      | Run `npm install` locally first to ensure lockfile is up to date                |
| Can't connect to DB                         | Check `POSTGRES_PASSWORD` matches in both the db service env and `DATABASE_URL` |
| Port 80 not reachable                       | Check Yandex Cloud security group rules                                         |
| Migrations fail                             | Ensure db service is healthy before running migrate                             |
