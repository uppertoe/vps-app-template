perl -0pi -e 's{(    restart: unless-stopped\n)}{$1    ports:\n      - "8080:8080"\n}' deploy/apps/*/docker-compose.yml
