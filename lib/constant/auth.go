package constant

import (
	"erp-cbqa-global/lib/encrypt"
)

func AuthDistributionCenter(bearerToken string) string {
	tokenRaw, claims, err := encrypt.Parse(bearerToken)
	if err != nil || claims["RoleID"] == nil {
		return tokenRaw + " not found"
	}
	sp, ok := claims["MdDistributionCenterId"].(string)
	if !ok {
		return ""
	}
	return sp
}

func AuthStockPoint(bearerToken string) string {
	tokenRaw, claims, err := encrypt.Parse(bearerToken)
	if err != nil || claims["RoleID"] == nil {
		return tokenRaw + " not found"
	}
	sp, ok := claims["MdStockpointId"].(string)
	if !ok {
		return ""
	}
	return sp
}
