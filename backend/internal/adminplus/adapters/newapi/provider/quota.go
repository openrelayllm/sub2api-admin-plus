package provider

import "math"

const newAPIQuotaUnitsPerUSD = 500000.0

func newAPIQuotaToUSDAmount(rawQuota float64) float64 {
	if rawQuota <= 0 {
		return 0
	}
	return rawQuota / newAPIQuotaUnitsPerUSD
}

func newAPIQuotaToUSDCents(rawQuota float64) int64 {
	return int64(math.Round(newAPIQuotaToUSDAmount(rawQuota) * 100))
}

func usdAmountToNewAPIQuotaUnits(amount float64) int64 {
	if amount <= 0 {
		return 0
	}
	return int64(math.Round(amount * newAPIQuotaUnitsPerUSD))
}
