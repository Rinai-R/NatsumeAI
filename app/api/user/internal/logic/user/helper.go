package logic

import (
	"NatsumeAI/app/api/user/internal/types"
	"NatsumeAI/app/services/user/userservice"
)


func toUserProfile(u *userservice.UserProfile) types.UserProfile {
	if u == nil {
		return types.UserProfile{}
	}

	return types.UserProfile{
		UserId:    u.UserId,
		Username:  u.Username,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}