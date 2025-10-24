CREATE TABLE IF NOT EXISTS `coupons` (
    `id`               BIGINT           NOT NULL AUTO_INCREMENT COMMENT '券模板ID',
    `coupon_type`      TINYINT          NOT NULL COMMENT '券类型 1-现金券 2-折扣券 3-免邮券',
    `discount_amount`  BIGINT           NOT NULL DEFAULT 0 COMMENT '抵扣金额(分)，折扣券为上限',
    `discount_percent` BIGINT           NOT NULL DEFAULT 0 COMMENT '折扣百分比',
    `min_spend_amount` BIGINT           NOT NULL DEFAULT 0 COMMENT '使用门槛(分)',
    `total_quantity`   BIGINT           NOT NULL DEFAULT 0 COMMENT '发券总量(0不限)',
    `per_user_limit`   BIGINT           NOT NULL DEFAULT 0 COMMENT '每人限领数量(0不限)',
    `issued_quantity`  BIGINT           NOT NULL DEFAULT 0 COMMENT '已发放数量',
    `start_at`         DATETIME         NOT NULL COMMENT '有效期开始时间',
    `end_at`           DATETIME         NOT NULL COMMENT '有效期结束时间',
    `source`           VARCHAR(64)      NOT NULL DEFAULT '' COMMENT '来源',
    `remarks`          VARCHAR(255)     NOT NULL DEFAULT '' COMMENT '备注',
    `created_at`       DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`       DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_time_range` (`start_at`,`end_at`)
);

CREATE TABLE IF NOT EXISTS `coupon_instances` (
    `id`              BIGINT NOT NULL AUTO_INCREMENT COMMENT '券实例ID',
    `coupon_id`       BIGINT NOT NULL COMMENT '对应的优惠券id',
    `user_id`         BIGINT NOT NULL COMMENT '持券用户ID',
    `status`          ENUM('UNUSED','LOCKED','USED', 'EXPIRED') NOT NULL DEFAULT 'UNUSED' COMMENT '实例状态',
    `locked_preorder` BIGINT NOT NULL DEFAULT 0 COMMENT '锁定的预订单ID',
    `locked_at`       DATETIME        NULL COMMENT '锁定时间',
    `used_order_id`   BIGINT NOT NULL DEFAULT 0 COMMENT '使用的订单ID',
    `used_at`         DATETIME        NULL COMMENT '使用时间',
    `created_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_user_status` (`user_id`,`status`)
);
