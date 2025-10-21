package errno

const (
	StatusOK           = 10000
	StatusTokenFreshed = 10001
)

const (
	TokenEmpty = 40000 + iota
	AccessTokenExpired
	RefreshTokenExpired
)

const (
	InternalError = 50000 + iota
	InvalidParam
	UserAlreadyExists
	UserNotFound
	InvalidCredentials
	AddressNotFound
	AddressForbidden
	ProductNotFound
	MerchantMismatch
)

const (
	InsertInventoryError = 60000 + iota
)
