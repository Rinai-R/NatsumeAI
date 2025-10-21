package inventory

import (
	"errors"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var ErrNotFound = sqlx.ErrNotFound
var ErrRowsAffectedIsZero = errors.New("affected rows is zero")
var ErrInvalidParam = errors.New("invalid param for sql")
var ErrInsertError = errors.New("Insert inventory error")