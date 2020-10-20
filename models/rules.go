package models

// Point
func VoteAndCommentRule(amount, userId string) bool {
	return true
}

func GetvoteRule(amount, userId string) bool {
	return true
}

func FirstPublishedDailyRule(amount, userId string) bool {
	return true
}

func GroupActivityRule(amount, userId string) bool {
	return true
}

func NodeRunningRule(amount, userId string) bool {
	return true
}

func RoundtableGovernanceRule(amount, userId string) bool {
	return true
}

// WGT: Mining based on points only
func GrowthLimitRule(amount, userId string) bool {
	return true
}
