CREATE TABLE IF NOT EXISTS `inventory` (
    `product_id` BIGINT NOT NULL COMMENT '对应的商品id',
    `merchant_id` BIGINT NOT NULL COMMENT '商家id',
    `stock` BIGINT NOT NULL COMMENT '现有可售库存',
    `sold` BIGINT NOT NULL COMMENT '已经售出的商品数量',
    `forzen_stock` BIGINT NOT NULL COMMENT '冻结库存',
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`product_id`)
);

CREATE TABLE IF NOT EXISTS `inventory_audit` (
  `id`         BIGINT NOT NULL AUTO_INCREMENT COMMENT '审计id',
  `order_id`   BIGINT NOT NULL COMMENT '对应的订单id',
  `product_id`     BIGINT NOT NULL COMMENT '对应的商品id',
  `quantity`   BIGINT NOT NULL COMMENT '商品的数量',
  `status`     ENUM('PENDING','CONFIRMED','CANCELLED') NOT NULL DEFAULT 'PENDING' COMMENT '库存状态，和库存原子更新',

  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY `uniq_order_product` (`order_id`, `product_id`),
  KEY `idx_product_status` (`product_id`, `status`),
  KEY `idx_status` (`status`),
  PRIMARY KEY (`id`)
);
