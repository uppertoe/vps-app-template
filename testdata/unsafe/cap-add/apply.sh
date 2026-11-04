perl -0pi -e 's{(    cap_drop:\n      - ALL\n)}{$1    cap_add:\n      - SYS_ADMIN\n}' deploy/apps/*/docker-compose.yml
