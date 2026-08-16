package listingcalc

import (
	"errors"
	"math"
)

var ErrInvalidMoney = errors.New("money values must be non-negative integer cents")

// ProfitCents calculates contribution profit without ever converting money to floats.
func ProfitCents(salePrice, purchaseCost, shippingCost, platformFee, otherCost int64) (int64, error) {
	if salePrice < 0 || purchaseCost < 0 || shippingCost < 0 || platformFee < 0 || otherCost < 0 {
		return 0, ErrInvalidMoney
	}
	return salePrice - purchaseCost - shippingCost - platformFee - otherCost, nil
}

// BasisPoints returns a ratio in 1/100 of a percent, rounded half up.
func BasisPoints(numerator, denominator int64) (int64, error) {
	if numerator < 0 || denominator <= 0 || numerator > math.MaxInt64/10000 {
		return 0, ErrInvalidMoney
	}
	return (numerator*10000 + denominator/2) / denominator, nil
}
