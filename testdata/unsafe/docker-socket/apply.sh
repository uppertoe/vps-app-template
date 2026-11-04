perl -0pi -e 's{(      - \S+_data:/data\n)}{$1      - /var/run/docker.sock:/var/run/docker.sock\n}' deploy/apps/*/docker-compose.yml
