#!/bin/bash
# shellcheck disable=SC2034
# shellcheck disable=SC2046
# shellcheck disable=SC2086

PG_USER=$POSTGRES_USER
PG_PASSWORD=$POSTGRES_PASSWORD
PG_DB=$POSTGRES_DB_NAME
PG_HOST="localhost"
PG_PORT=5432
PG_SSL_MODE="disable"

BACKUP_DIR="$HOME/go/src/go-ai/.files/bkp" # Replace with your desired backup directory
BACKUP_FILE="${BACKUP_DIR}/${PG_DB}_backup_$(date +%Y%m%d%H%M%S).sql" # Backup file name with timestamp

mkdir -p "$BACKUP_DIR"

# export PGPASSWORD=$PG_PASSWORD && pg_dump -U "$PG_USER" -h $PG_HOST -p $PG_PORT -d "$PG_DB" -F c -b -v -f "$BACKUP_FILE" --no-password

CMD1="pg_dump -U $PG_USER -h $PG_HOST -p $PG_PORT -d $PG_DB -F c -b -v"
CMD2="pg_dump -U $PG_USER -h $PG_HOST -p $PG_PORT -d $PG_DB -v"
CMD3=${CMD2}" --inserts"

docker run --rm \
  -e PGPASSWORD="$PG_PASSWORD" \
  -v $BACKUP_DIR:/backup \
  artifactory.dev.ict/docker-virtual/postgres:latest \
  $CMD3 -f /backup/$(basename $BACKUP_FILE)

echo "Backup of database '$PG_DB' completed successfully at $BACKUP_FILE"



# pg_dump -U postgres -C  -f backup_2022_12_06.dump -Fc treebase
# pg_restore  -U postgres -C -dpostgres -Fc backup_2022_12_06.dump