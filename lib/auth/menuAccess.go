package auth

import "strings"

func CheckIgnoreAPI(url, urlPath string, length int) bool {
	if length == 0 || strings.Contains(url, "is_all=true") || (urlPath == "/dashboard/auth/logout") {
		return true
	}
	return false
}

func FullCheck(pathAction string) bool {
	return pathAction == "role"
}

func GetMenuPath(menu string) string {
	switch menu {
	case "Outlet Order":
		return "outlet-order"
	case "User Management":
		return "user"
	case "Role Management":
		return "role"
	case "Loading Sheet":
		return "loading-sheet"
	case "Admin Management":
		return "admin-management"
	case "Driver":
		return "driver"
	case "Vehicle":
		return "vehicle"
	case "Purchase Order":
		return "purchase"
	case "Delivery From Vendor":
		return "delivery-igr"
	case "Delivery From DC":
		return "delivery-dc"
	case "Delivery From SP":
		return "delivery-sp"
	case "Delivery Outlet":
		return "delivery-outlet"
	case "Delivery Return":
		return "delivery-return"
	case "Refund":
		return "refund"
	case "Outlet":
		return "outlet"
	case "Sortation at DC ":
		return "sortation-dc"
	case "Report":
		return "report"
	case "RRP":
		return "rrp"
	case "Rekonsiliasi":
		return "rekon"
	}
	return "false"
}
