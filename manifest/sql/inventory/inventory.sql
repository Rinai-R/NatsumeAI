CREATE TABLE IF NOT EXISTS `inventory` (
    `product_id` BIGINT NOT NULL COMMENT '对应的商品id',
    `stock` BIGINT NOT NULL COMMENT '现有可售库存',
    `sold` BIGINT NOT NULL COMMENT '已经售出的商品数量',
    `forzen_stock` BIGINT NOT NULL COMMENT '冻结库存',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`product_id`)
);