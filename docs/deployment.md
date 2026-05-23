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

Estimated cost: ~1000–1500 RUB/month.

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
  --ssh-key ~/.ssh/id_rsa.pub
```

Get the external IP:

```bash
yc compute instance list
```

---

## 2. Open Firewall Ports

### Yandex Cloud Security Group (cloud-level)

In the console: **Virtual Private Cloud → Security Groups → your group → Add rule**

Add inbound rules for:

- TCP port 22 (SSH)
- TCP port 80 (HTTP)

Or via CLI:

```bash
yc vpc security-group update-rules <SECURITY_GROUP_ID> \
  --add-rule "direction=ingress,port=22,protocol=tcp,v4-cidr-blocks=0.0.0.0/0" \
  --add-rule "direction=ingress,port=80,protocol=tcp,v4-cidr-blocks=0.0.0.0/0"
```

### VM-level firewall (ufw)

SSH into the VM first, then:

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw enable
```

**Important:** Add both rules before running `ufw enable` — otherwise you may lose your SSH session.

---

## 3. SSH Into the VM

```bash
ssh -i ~/.ssh/<YOUR_KEY> yc-user@<VM_IP>
```

`yc-user` is the default username for Yandex Cloud Ubuntu images.

---

## 4. Install Docker

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

# Allow running docker without sudo
sudo usermod -aG docker $USER
newgrp docker

# Verify
docker --version
docker compose version
```

---

## 5. Clone the Repository

```bash
cd ~
git clone https://github.com/<YOUR_USER>/shmanki.git
cd shmanki
```

---

## 6. Configure Environment

Create `.env.prod` in the project root. **This file is never committed — create it manually on the VM.**

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

## 7. First Deploy

```bash
cd ~/shmanki

# Build and start all services (db, backend, frontend, nginx)
docker compose -f compose.prod.yaml --env-file .env.prod up -d --build

# Run database migrations
docker compose -f compose.prod.yaml --env-file .env.prod --profile tools run --rm migrate

# Check everything is running
docker compose -f compose.prod.yaml ps
```

The app is now available at `http://<VM_IP>`.

Check logs if something is wrong:

```bash
docker compose -f compose.prod.yaml logs -f
```

---

## 8. Redeployment

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

Save this to `~/redeploy.sh` on the VM (run once):

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

## 9. Useful Commands

```bash
# View live logs
docker compose -f compose.prod.yaml logs -f
docker compose -f compose.prod.yaml logs -f backend
docker compose -f compose.prod.yaml logs -f frontend

# Restart a single service
docker compose -f compose.prod.yaml restart backend

# Open a DB shell
docker compose -f compose.prod.yaml exec db psql -U postgres -d shmanki

# Stop everything (keeps data)
docker compose -f compose.prod.yaml down

# Stop and wipe DB volume — destroys all data
docker compose -f compose.prod.yaml down -v
```

---

## 10. Project Files Relevant to Deployment

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

| Issue                            | Fix                                                                                            |
| -------------------------------- | ---------------------------------------------------------------------------------------------- |
| Can't SSH after `ufw enable`     | Use Yandex Cloud serial console to access the VM: **Compute Cloud → VM → Serial console**      |
| Port 80 unreachable              | Check Yandex Cloud Security Group has inbound TCP 80 rule                                      |
| Backend crashes on start         | Check `.env.prod` exists and `DATABASE_URL` uses `db` as hostname, not `localhost`             |
| `migrate` fails                  | Ensure db container is healthy: `docker compose -f compose.prod.yaml ps`                       |
| `build.server` fails in frontend | Run `npm install` locally, commit updated `package-lock.json`                                  |
| DB password mismatch             | `POSTGRES_PASSWORD` in `.env.prod` must be identical in both the db section and `DATABASE_URL` |
