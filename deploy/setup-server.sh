#!/bin/bash
set -e

SERVER_IP="47.76.151.183"
SERVER_USER="root"

echo "=== Server Initial Setup ==="

ssh ${SERVER_USER}@${SERVER_IP} << 'EOF'
echo "Updating system packages..."
apt-get update && apt-get upgrade -y

echo "Installing Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker
    systemctl start docker
fi

echo "Installing Docker Compose plugin..."
apt-get install -y docker-compose-plugin

echo "Creating deploy directory..."
mkdir -p /opt/aidevclub

echo "Docker version:"
docker --version
docker compose version

echo "=== Server setup completed! ==="
EOF
