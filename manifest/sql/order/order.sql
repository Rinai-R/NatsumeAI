CREATE TABLE IF NOT EXISTS `order_preorders` (
    `preorder_id`       BIGINT NOT NULL AUTO_INCREMENT COMMENT '预订单ID',
    `user_id`           BIGINT NOT NULL COMMENT '用户ID',
    `coupon_id`         BIGINT NOT NULL COMMENT '优惠券ID',
    `original_amount`   BIGINT NOT NULL COMMENT '原始金额',
    `final_amount`      BIGINT NOT NULL COMMENT '最终金额',
    `status`            ENUM('PENDING','READY','PLACED','CANCELLED') NOT NULL DEFAULT 'PENDING' COMMENT '状态：待处理/就绪/已下单/取消',
    `expire_at`         DATETIME        NOT NULL COMMENT '预订单过期时间',
    `created_at`        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`preorder_id`),
    KEY `idx_user_status` (`user_id`,`status`),
    KEY `idx_expire_at` (`expire_at`)
);

CREATE TABLE IF NOT EXISTS `order_preorder_items` (
    `id`            BIGINT NOT NULL AUTO_INCREMENT,
    `preorder_id`   BIGINT NOT NULL COMMENT '预订单ID',
    `product_id`    BIGINT NOT NULL COMMENT '商品ID',
    `quantity`      BIGINT NOT NULL COMMENT '商品数量',
    `price_cents`   BIGINT NOT NULL COMMENT '结账的快照单价(分)',
    `snapshot`      JSON             NULL COMMENT '商品的各种信息',
    `created_at`    DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_preorder` (`preorder_id`)
);

CREATE TABLE IF NOT EXISTS `orders` (
    `order_id`          BIGINT NOT NULL AUTO_INCREMENT COMMENT '订单ID',
    `preorder_id`       BIGINT NOT NULL COMMENT '预订单ID',
    `user_id`           BIGINT NOT NULL COMMENT '用户ID',
    `coupon_id`         BIGINT NOT NULL COMMENT '优惠券ID',
    `status`            ENUM('PENDING_PAYMENT','PAYING','PAID','CANCELLED','COMPLETED','REFUNDED') NOT NULL DEFAULT 'PENDING_PAYMENT' COMMENT '订单状态',
    

    `total_amount`      BIGINT NOT NULL DEFAULT 0 COMMENT '订单商品总金额(分)',
    `payable_amount`    BIGINT NOT NULL DEFAULT 0 COMMENT '应付金额(分)',
    `paid_amount`       BIGINT NOT NULL DEFAULT 0 COMMENT '实际支付金额(分)',


    `payment_method`    VARCHAR(32)     NOT NULL DEFAULT '' COMMENT '支付方式',
    `payment_at`        DATETIME        NULL COMMENT '支付时间',

    `expire_time`     BIGINT       NOT NULL COMMENT '过期时间戳',
    `cancel_reason`     VARCHAR(255)    NOT NULL DEFAULT '' COMMENT '取消原因',


    `address_snapshot`  JSON            NULL COMMENT '收货地址快照',
    `created_at`        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`order_id`),
    UNIQUE KEY `uk_preorder_id` (`preorder_id`),
    KEY `idx_user_status` (`user_id`,`status`),
    KEY `idx_expire_time` (`expire_time`)
);

CREATE TABLE IF NOT EXISTS `order_items` (
    `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `order_id`      BIGINT UNSIGNED NOT NULL COMMENT '订单ID',
    `product_id`    BIGINT UNSIGNED NOT NULL COMMENT '商品ID',
    `quantity`      BIGINT UNSIGNED NOT NULL COMMENT '商品数量',
    `price_cents`   BIGINT UNSIGNED NOT NULL COMMENT '下单时单价(分)',
    `snapshot`      JSON     NULL COMMENT '规格属性快照',
    `created_at`    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_order` (`order_id`)
);
