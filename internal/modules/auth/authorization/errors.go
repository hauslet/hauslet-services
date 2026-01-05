package authorization

import "errors"

var (
	ErrSupplyProfileRequired      = errors.New("profile required for supply access")
	ErrSupplyUserTypeRequired     = errors.New("supply user type required")
	ErrSupplyVerificationRequired = errors.New("identity verification required")
	ErrSupplySubscriptionRequired = errors.New("supply subscription required")
	ErrSupplyUnauthorized         = errors.New("unauthorized")
)
