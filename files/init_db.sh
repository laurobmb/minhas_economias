#!/bin/bash

ls -l /tmp/database
rm -rf /tmp/database
mkdir /tmp/database
podman run -it --rm --name minhas_economias-db -e POSTGRES_DB=minhas_economias -e POSTGRES_USER=me -e POSTGRES_PASSWORD=1q2w3e -p 5432:5432 --log-level=debug -v /tmp/database:/var/lib/postgresql/:Z postgres:latest
