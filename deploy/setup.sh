#!/bin/bash
# Run this script on a fresh Ubuntu 22.04 GCP VM to set up and start the XO backend.
# Usage: bash setup.sh

set -e

echo "=== Installing Docker ==="
sudo apt-get update -y
sudo apt-get install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update -y
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Allow current user to run docker without sudo
sudo usermod -aG docker "$USER"

echo "=== Docker installed ==="
docker --version
docker compose version

echo ""
echo "=== Cloning repo ==="
cd ~
git clone https://github.com/Kroszborg/xo.git
cd xo

echo ""
echo "=== Creating .env ==="
JWT_SECRET=$(openssl rand -hex 32)
cat > .env <<EOF
JWT_SECRET=${JWT_SECRET}
CORS_ORIGINS=*
EOF
echo ".env created with JWT_SECRET=${JWT_SECRET}"
echo "Save this secret somewhere safe!"

echo ""
echo "=== Starting stack (first boot pulls Ollama model ~2.5GB, takes 5-10 min) ==="
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

echo ""
echo "=== Done! Check status with: docker compose ps ==="
echo "Gateway API is on port 8000"
echo "Test: curl http://localhost:8000/health"
