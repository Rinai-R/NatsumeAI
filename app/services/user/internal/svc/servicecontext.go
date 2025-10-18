package svc

import (
	"NatsumeAI/app/common/consts/biz"
	usermodel "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/auth/auth"
	"NatsumeAI/app/services/auth/authservice"
	"NatsumeAI/app/services/user/internal/config"
	"context"

	"github.com/zeromicro/go-zero/core/bloom"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config config.Config
	AuthRpc auth.AuthServiceClient

	UserModel usermodel.UsersModel
	UserAddressModel usermodel.UserAddressesModel
	Bloom *bloom.Filter
}

func NewServiceContext(c config.Config) *ServiceContext {
	bf := bloom.New(redis.MustNewRedis(c.RedisConf),  biz.USER_LOGIN_BLOOM, biz.USER_LOGIN_BLOOM_BIT)
	UserModel := usermodel.NewUsersModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf)
	bloomPreheat(bf, UserModel)
	return &ServiceContext{
		Config: c,
		AuthRpc: authservice.NewAuthService(zrpc.MustNewClient(c.AuthRpc)),
		UserModel: UserModel,
		UserAddressModel: usermodel.NewUserAddressesModel(sqlx.MustNewConn(c.MysqlConf), c.CacheConf),
		Bloom: bf,
	}
}


func bloomPreheat(bf *bloom.Filter, UsersModel usermodel.UsersModel) error {

	names, err := UsersModel.FindAllUsername(context.TODO())
	if err != nil {
		return err
	}

	for _, names := range names {
		err := bf.Add([]byte(names))
		if err != nil {
			return err
		}
	}
	return nil

}
