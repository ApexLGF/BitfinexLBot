#!/bin/bash
# 创建多个数据库的脚本

set -e
set -u

function create_user_and_database() {
    local database=$1
    echo "  Creating user and database '$database'"
    mysql -u root -p$MYSQL_ROOT_PASSWORD <<-EOSQL
        CREATE DATABASE IF NOT EXISTS \`$database\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
EOSQL
}

if [ -n "$MYSQL_MULTIPLE_DATABASES" ]; then
    echo "Multiple database creation requested: $MYSQL_MULTIPLE_DATABASES"
    for db in $(echo $MYSQL_MULTIPLE_DATABASES | tr ',' ' '); do
        create_user_and_database $db
    done
    echo "Multiple databases created"
fi

# 为 BitfinexBot 创建专用用户
echo "Creating BitfinexBot database user..."
mysql -u root -p$MYSQL_ROOT_PASSWORD <<-EOSQL
    CREATE USER IF NOT EXISTS 'bitfinex_bot'@'%' IDENTIFIED BY 'BitfinexBot@2025';
    GRANT ALL PRIVILEGES ON bitfinex_bot_db.* TO 'bitfinex_bot'@'%';
    FLUSH PRIVILEGES;
EOSQL

echo "BitfinexBot database user created successfully"