#!/bin/sh
sudo -u postgres createdb acmedns
sudo -u postgres psql -c "CREATE USER acmedns WITH PASSWORD 'acmedns'"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE acmedns TO acmedns"
