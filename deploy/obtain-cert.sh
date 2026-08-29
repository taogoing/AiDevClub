#!/bin/bash
set -e

cd /opt/aidevclub

DOMAIN="aidevclub.xyz"
EMAIL="${CERTBOT_EMAIL:-admin@aidevclub.xyz}"
VOLUME="aidevclub_certbot_certs"

certbot_tracks() {
  docker compose run --rm --entrypoint certbot certbot certificates 2>/dev/null | grep -q "${DOMAIN}"
}

echo "=== Ensuring services are up ==="
docker compose up -d

if certbot_tracks; then
  echo "=== certbot certificates ==="
  docker compose run --rm --entrypoint certbot certbot certificates
  echo "=== nginx config test ==="
  docker compose exec frontend nginx -t
  echo "=== cert files ==="
  docker compose exec frontend ls -la /etc/letsencrypt/live/aidevclub.xyz/ 2>&1
  echo "=== TLS test from inside network ==="
  docker run --rm --network aidevclub_default alpine sh -c "command -v openssl >/dev/null 2>&1 || apk add --no-cache openssl >/dev/null 2>&1; echo | openssl s_client -connect frontend:443 -servername aidevclub.xyz 2>&1 | head -25"
  echo "=== Reloading nginx ==="
  docker compose exec frontend nginx -s reload || true
  echo "=== service status ==="
  docker compose ps
  exit 0
fi

echo "=== Creating fallback self-signed cert (so nginx can start before the real cert exists) ==="
docker run --rm -v "${VOLUME}:/etc/letsencrypt" alpine sh -c "
  command -v openssl >/dev/null 2>&1 || (apk update >/dev/null 2>&1 && apk add --no-cache openssl >/dev/null 2>&1)
  mkdir -p /etc/letsencrypt/live/${DOMAIN}
  if [ ! -f /etc/letsencrypt/live/${DOMAIN}/fullchain.pem ]; then
    openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
      -keyout /etc/letsencrypt/live/${DOMAIN}/privkey.pem \
      -out /etc/letsencrypt/live/${DOMAIN}/fullchain.pem \
      -subj \"/CN=${DOMAIN}\"
  fi
"

echo "=== Restarting frontend with fallback cert ==="
docker compose restart frontend
sleep 3

echo "=== Requesting Let's Encrypt certificate ==="
# certbot refuses to create a live/ dir that already exists (e.g. our fallback
# self-signed cert), so remove the fallback dir before requesting the real one.
docker run --rm -v "${VOLUME}:/etc/letsencrypt" alpine sh -c \
  "rm -rf /etc/letsencrypt/live/${DOMAIN} /etc/letsencrypt/archive/${DOMAIN} /etc/letsencrypt/renewal/${DOMAIN}.conf"

docker compose run --rm --entrypoint certbot certbot certonly --webroot \
  --webroot-path /var/www/certbot \
  -d "${DOMAIN}" -d "www.${DOMAIN}" \
  --non-interactive --agree-tos --email "${EMAIL}"

echo "=== Reloading nginx to use the real certificate ==="
docker compose exec frontend nginx -s reload

echo "=== TLS setup complete ==="
