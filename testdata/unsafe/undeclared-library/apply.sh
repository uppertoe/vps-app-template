perl -0pi -e 's{(\tgopkg\.in/yaml\.v3 v3\.0\.1\n)}{$1\tgithub.com/google/uuid v1.6.0\n}' go.mod
