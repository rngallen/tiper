package auth

import "dfms/pkg/response"

func stableOrder(column, dir string) string {
	return response.StableOrder(column, dir)
}
