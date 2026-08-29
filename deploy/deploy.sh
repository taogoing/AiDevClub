#!/bin/bash
set -e

SERVER_IP="47.76.151.183"
SERVER_USER="root"
DEPLOY_DIR="/opt/aidevclub"

echo "=== AiDevClub Deployment Script ==="

if [ ! -f "deploy/.env" ]; then
    echo "Error: deploy/.env not found!"
    echo "Please copy deploy/.env.example to deploy/.env and configure it."
    exit 1
fi

echo "Step 1: Building Docker images locally..."
docker build -t aidevclub-backend -f deploy/Dockerfile.backend .
docker build -t aidevclub-frontend -f deploy/Dockerfile.frontend .

echo "Step 2: Saving Docker images..."
mkdir -p /tmp/aidevclub-deploy
docker save aidevclub-backend | gzip > /tmp/aidevclub-deploy/backend.tar.gz
docker save aidevclub-frontend | gzip > /tmp/aidevclub-deploy/frontend.tar.gz

echo "Step 3: Copying files to server..."
ssh ${SERVER_USER}@${SERVER_IP} "mkdir -p ${DEPLOY_DIR}"

scp /tmp/aidevclub-deploy/backend.tar.gz ${SERVER_USER}@${SERVER_IP}:${DEPLOY_DIR}/
scp /tmp/aidevclub-deploy/frontend.tar.gz ${SERVER_USER}@${SERVER_IP}:${DEPLOY_DIR}/
scp deploy/docker-compose.prod.yml ${SERVER_USER}@${SERVER_IP}:${DEPLOY_DIR}/docker-compose.yml
scp deploy/.env ${SERVER_USER}@${SERVER_IP}:${DEPLOY_DIR}/.env

echo "Step 4: Deploying on server..."
ssh ${SERVER_USER}@${SERVER_IP} << 'EOF'
cd /opt/aidevclub

echo "Loading Docker images..."
docker load -i backend.tar.gz
docker load -i frontend.tar.gz

echo "Starting services..."
docker compose down || true
docker compose up -d

echo "Waiting for services to be healthy..."
sleep 10

echo "Requesting TLS certificate (if not already present)..."
CERT_EMAIL="${CERTBOT_EMAIL:-admin@aidevclub.xyz}"
if ! docker compose run --rm --entrypoint test certbot -f /etc/letsencrypt/live/aidevclub.xyz/fullchain.pem; then
  docker compose run --rm certbot certonly --webroot \
    --webroot-path /var/www/certbot \
    -d aidevclub.xyz -d www.aidevclub.xyz \
    --non-interactive --agree-tos --email "${CERT_EMAIL}"
  docker compose restart frontend
else
  echo "Certificate already exists, skipping issuance."
fi

echo "Checking service status..."
docker compose ps

echo "Cleaning up..."
rm -f backend.tar.gz frontend.tar.gz
EOF

echo "Step 5: Cleaning up local files..."
rm -rf /tmp/aidevclub-deploy

echo "=== Deployment completed! ==="
echo "Visit: http://${SERVER_IP}"
