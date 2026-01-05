package authorization

import "errors"

var (
	ErrSupplyProfileRequired      = errors.New("profile required for supply access")
	ErrSupplyUserTypeRequired     = errors.New("supply user type required")
	ErrSupplyVerificationRequired = errors.New("identity verification required")
	ErrSupplySubscriptionRequired = errors.New("supply subscription required")
	ErrSupplyUnauthorized         = errors.New("unauthorized")

	ErrResourceProfileRequired             = errors.New("profile required for resource access")
	ErrResourceEmailRequired               = errors.New("email required for resource access")
	ErrResourcePhoneRequired               = errors.New("phone number required for resource access")
	ErrResourcePhoneVerificationRequired   = errors.New("phone verification required for resource access")
	ErrResourceIDVerificationRequired      = errors.New("identity verification required for resource access")
	ErrResourceVerificationLevelRequired   = errors.New("verification level required for resource access")
	ErrResourceUnauthorized                = errors.New("unauthorized")
)
