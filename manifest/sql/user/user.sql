CREATE TABLE IF NOT EXISTS `users` (
    `id`            BIGINT UNSIGNED NOT NULL COMMENT '唯一主键，雪花执行',
    `username`      VARCHAR(64) NOT NULL COMMENT '唯一的用户名',
    `password`      VARCHAR(255) NOT NULL COMMENT '密码这块，肯定是加密过的密码',
    `created_at`    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最近更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_users_username` (`username`)
);

CREATE TABLE IF NOT EXISTS `user_addresses` (
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键，搞个聚簇索引',
    `user_id`    BIGINT UNSIGNED NOT NULL COMMENT '关联的用户id',
    `detail`     VARCHAR(255) NOT NULL COMMENT '懒得写太多crud，直接一个detail',
    `is_default` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是不是一个默认地址',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最近更新时间',
    PRIMARY KEY (`id`),
    KEY `idx_user_addresses_user_id` (`user_id`)
);


CREATE TABLE IF NOT EXISTS `merchants` (
    `id`                         BIGINT          NOT NULL AUTO_INCREMENT COMMENT '商家ID',
    `user_id`                    BIGINT UNSIGNED NOT NULL COMMENT '申请人用户ID（users.id）',
    `shop_name`                  VARCHAR(128)    NOT NULL COMMENT '店铺名称',
    `contact_name`               VARCHAR(64)     NOT NULL DEFAULT '' COMMENT '联系人姓名',
    `contact_phone`              VARCHAR(32)     NOT NULL DEFAULT '' COMMENT '联系人手机号',
    `address`                    VARCHAR(255)    NOT NULL DEFAULT '' COMMENT '经营地址',
    `description`                TEXT            NULL COMMENT '其他补充说明',
    `status`                     ENUM('PENDING','APPROVED','REJECTED','ESCALATED') NOT NULL DEFAULT 'PENDING' COMMENT '申请状态',
    `reject_reason`              VARCHAR(255)    NOT NULL DEFAULT '' COMMENT '驳回原因',
    `reviewed_at`                DATETIME        NULL DEFAULT NULL COMMENT '审核时间',
    `created_at`                 TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at`                 TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最近更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_merchants_user_id` (`user_id`),
    KEY `idx_merchants_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
