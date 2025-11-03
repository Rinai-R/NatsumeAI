CREATE TABLE IF NOT EXISTS `payment_orders` (
    `payment_id`      BIGINT NOT NULL AUTO_INCREMENT COMMENT '支付记录ID',
    `payment_no`      VARCHAR(64) NOT NULL COMMENT '支付单号',
    `order_id`        BIGINT NOT NULL COMMENT '关联订单ID',
    `user_id`         BIGINT NOT NULL COMMENT '用户ID',
    `amount`          BIGINT NOT NULL COMMENT '支付金额，单位分',
    `currency`        VARCHAR(8) NOT NULL DEFAULT 'CNY' COMMENT '币种',
    `channel`         VARCHAR(32) NOT NULL COMMENT '支付渠道',
    `status`          ENUM('INIT','PROCESSING','SUCCESS','FAILED','CANCELLED','EXPIRED') NOT NULL DEFAULT 'INIT' COMMENT '支付状态',
    `channel_payload` JSON NULL COMMENT '渠道请求载荷',
    `timeout_at`      DATETIME NOT NULL COMMENT '支付超时时间',
    `extra`           JSON NULL COMMENT '附加信息',
    `created_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`payment_id`),
    UNIQUE KEY `uk_payment_no` (`payment_no`),
    UNIQUE KEY `uk_order_id` (`order_id`),
    KEY `idx_user_status` (`user_id`,`status`)
);
