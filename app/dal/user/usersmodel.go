package user

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UsersModel = (*customUsersModel)(nil)

type (
	// UsersModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUsersModel.
	UsersModel interface {
		usersModel
		FindAllUsername(ctx context.Context) ([]string, error) 
	}

	customUsersModel struct {
		*defaultUsersModel
	}
)

// NewUsersModel returns a model for the database table.
func NewUsersModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UsersModel {
	return &customUsersModel{
		defaultUsersModel: newUsersModel(conn, c, opts...),
	}
}

func (m *customUsersModel) FindAllUsername(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf("SELECT username FROM %s", m.table)
	var name []string
	err := m.CachedConn.QueryRowNoCacheCtx(ctx, &name, query)
	return name, err
}