FROM php:8.1-fpm-alpine

# 安装必要的包和PHP扩展
RUN apk add --no-cache \
    mysql-client \
    && docker-php-ext-install pdo pdo_mysql mysqli

# 设置工作目录
WORKDIR /var/www/html

# 复制PHP配置（如果有的话）
# COPY php.ini /usr/local/etc/php/

EXPOSE 9000

CMD ["php-fpm"]