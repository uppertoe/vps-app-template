perl -0pi -e 's{import role "[^"]+" "[^"]+" (\S+)}{reverse_proxy $1}' deploy/apps/*/*.caddy
