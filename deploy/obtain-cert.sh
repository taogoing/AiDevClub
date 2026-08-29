#!/bin/bash
set -e

cd /opt/aidevclub

DOMAIN="aidevclub.xyz"
EMAIL="${CERTBOT_EMAIL:-admin@aidevclub.xyz}"
VOLUME="aidevclub_certbot_certs"

# Returns 0 only if a REAL Let's Encrypt certificate (not the self-signed fallback) is present.
real_cert_present() {
  docker compose exec -T frontend sh -c \
    "openssl x509 -in /etc/letsencrypt/live/${DOMAIN}/fullchain.pem -noout -issuer 2>/dev/null" \
    | grep -qi "Let's Encrypt"
}

echo "=== Ensuring services are up ==="
docker compose up -d

if real_cert_present; then
  echo "=== Valid Let's Encrypt certificate already present; reloading nginx ==="
  docker compose exec frontend nginx -s reload || true
  docker compose ps
  exit 0
fi

echo "=== No valid certificate yet; creating fallback self-signed cert so nginx can start ==="
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

echo "=== Removing fallback / stale cert data so certbot can issue a fresh certificate ==="
docker run --rm -v "${VOLUME}:/etc/letsencrypt" alpine sh -c \
  "rm -rf /etc/letsencrypt/live/${DOMAIN} /etc/letsencrypt/archive/${DOMAIN} /etc/letsencrypt/renewal/${DOMAIN}.conf"

echo "=== Requesting Let's Encrypt certificate ==="
docker compose run --rm --entrypoint certbot certbot certonly --webroot \
  --webroot-path /var/www/certbot \
  -d "${DOMAIN}" -d "www.${DOMAIN}" \
  --non-interactive --agree-tos --email "${EMAIL}"

echo "=== Reloading nginx to use the real certificate ==="
docker compose exec frontend nginx -s reload

echo "=== service status ==="
docker compose ps
echo "=== TLS verification ==="
docker run --rm --network aidevclub_default alpine sh -c "command -v openssl >/dev/null 2>&1 || apk add --no-cache openssl >/dev/null 2>&1; echo | openssl s_client -connect frontend:443 -servername ${DOMAIN} 2>&1 | grep -E 'subject=|issuer=|Verify'"

echo "=== TLS setup complete ==="
