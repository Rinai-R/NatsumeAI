package svc

import (
	couponmodel "NatsumeAI/app/dal/coupon"
	"NatsumeAI/app/services/coupon/internal/config"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config               config.Config
	MysqlConn            sqlx.SqlConn
	CouponsModel         couponmodel.CouponsModel
	CouponInstancesModel couponmodel.CouponInstancesModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)

	conn := sqlx.MustNewConn(c.MysqlConf)

	return &ServiceContext{
		Config:               c,
		MysqlConn:            conn,
		CouponsModel:         couponmodel.NewCouponsModel(conn, c.CacheConf),
		CouponInstancesModel: couponmodel.NewCouponInstancesModel(conn, c.CacheConf),
	}
}
