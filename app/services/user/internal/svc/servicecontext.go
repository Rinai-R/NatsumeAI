package svc

import (
	commoncfg "NatsumeAI/app/common/config"
	"NatsumeAI/app/common/consts/biz"
	usermodel "NatsumeAI/app/dal/user"
	"NatsumeAI/app/services/auth/auth"
	"NatsumeAI/app/services/auth/authservice"
	"NatsumeAI/app/services/user/internal/config"
	"context"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-zero/core/bloom"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config  config.Config
	AuthRpc auth.AuthServiceClient
	// Casbin enforcer for policy changes (e.g., bind roles)
	Casbin *casbin.DistributedEnforcer

	MysqlConn        sqlx.SqlConn
	UserModel        usermodel.UsersModel
	UserAddressModel usermodel.UserAddressesModel
	MerchantsModel   usermodel.MerchantsModel
	ChatModel        *ark.ChatModel
	Bloom            *bloom.Filter

	KafkaWriter *kafka.Writer
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.MustSetup(c.LogConf)
	bf := bloom.New(redis.MustNewRedis(c.RedisConf), biz.USER_LOGIN_BLOOM, biz.USER_LOGIN_BLOOM_BIT)
	conn := sqlx.MustNewConn(c.MysqlConf)
	userModel := usermodel.NewUsersModel(conn, c.CacheConf)
	bloomPreheat(bf, userModel)

	var kw *kafka.Writer
	if len(c.KafkaConf.Broker) > 0 && c.KafkaConf.MerchantReviewTopic != "" {
		kw = &kafka.Writer{
			Addr:                   kafka.TCP(c.KafkaConf.Broker...),
			Topic:                  c.KafkaConf.MerchantReviewTopic,
			RequiredAcks:           kafka.RequireOne,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
			BatchTimeout:           5 * time.Millisecond,
		}
	}
	// build casbin enforcer from config
	var enforcer *casbin.DistributedEnforcer
	if (commoncfg.CasbinMiddlewareConf{} != c.CasbinMiddleware) && c.CasbinMiddleware.Dns != "" && c.CasbinMiddleware.Model != "" {
		enforcer = c.CasbinMiddleware.MustNewDistributedEnforcer()
	}

	var chatModel *ark.ChatModel
	if c.ChatModel.BaseUrl != "" && c.ChatModel.APIKey != "" && c.ChatModel.Model != "" {
		cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
			BaseURL: c.ChatModel.BaseUrl,
			APIKey:  c.ChatModel.APIKey,
			Model:   c.ChatModel.Model,
		})
		if err != nil {
			logx.Errorw("init ark chat model failed", logx.Field("err", err))
		} else {
			chatModel = cm
			logx.Infow("ark chat model initialized")
		}
	}

	return &ServiceContext{
		Config:           c,
		AuthRpc:          authservice.NewAuthService(zrpc.MustNewClient(c.AuthRpc)),
		Casbin:           enforcer,
		MysqlConn:        conn,
		UserModel:        userModel,
		UserAddressModel: usermodel.NewUserAddressesModel(conn, c.CacheConf),
		MerchantsModel:   usermodel.NewMerchantsModel(conn, c.CacheConf),
		ChatModel:        chatModel,
		Bloom:            bf,
		KafkaWriter:      kw,
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
