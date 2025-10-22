CREATE TABLE IF NOT EXISTS `products` (
    `id`          BIGINT NOT NULL AUTO_INCREMENT COMMENT '商品主键',
    `merchant_id` BIGINT NOT NULL COMMENT '商家id',
    `name`        VARCHAR(128) NOT NULL COMMENT '商品名称',
    `description` TEXT NOT NULL COMMENT '商品描述',
    `picture`     VARCHAR(255) NOT NULL COMMENT '商品主图地址',
    `price`       BIGINT NOT NULL COMMENT '商品售价，单位分',
    `created_at`  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_products_updated_at` (`updated_at`),
    KEY `idx_products_price` (`price`)
);

CREATE TABLE IF NOT EXISTS `product_categories` (
    `id`         BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键',
    `product_id` BIGINT NOT NULL COMMENT '关联的商品id',
    `category`   VARCHAR(64) NOT NULL COMMENT '类目名称',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_product_categories_product_id_category` (`product_id`, `category`),
    KEY `idx_product_categories_category` (`category`)
);
