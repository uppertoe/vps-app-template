perl -0pi -e 's{(FROM golang:[0-9.]+-alpine)\@sha256:[0-9a-f]+}{$1}' Dockerfile
