perl -0pi -e 's{go build -trimpath}{go build -tags dev -trimpath}' Dockerfile
