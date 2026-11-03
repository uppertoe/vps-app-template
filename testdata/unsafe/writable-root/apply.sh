perl -0pi -e 's{read_only: true}{read_only: false}' deploy/apps/*/docker-compose.yml
