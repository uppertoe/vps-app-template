perl -0pi -e 's{import protected_groups "([^"]+)" (\S+)}{reverse_proxy $2}' deploy/apps/*/*.caddy
