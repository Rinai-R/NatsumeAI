CREATE TABLE IF NOT EXISTS `cart` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键，纯摆设',
    `product_id` BIGINT NOT NULL COMMENT '关联的商品id',
    `user_id` BIGINT NOT NULL COMMENT '关联的用户id',
    `quantity` BIGINT NOT NULL COMMENT '商品数量',
    PRIMARY KEY (`id`),
    KEY `idx_cart_user_product` (`user_id`, `product_id`)
);
