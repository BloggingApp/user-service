package redisrepo

import "fmt"

const (
	USER_KEY = "user:%s" // <userID>
	USER_BY_USERNAME_KEY = "user-name:%s" // <username>
	TEMP_REGISTRATION_CODE_KEY = "registration-code:%d" // <registration code>
	TEMP_SIGNIN_CODE_KEY = "sign-in-code:%d" // <sign-in code>
	SEARCH_RESULTS_KEY = "search-results:%s:%d:%d" // <any word>:<limit>:<offset>
	USER_FOLLOWERS_KEY = "user-followers:%s:%d:%d" // <userID>:<limit>:<offset>
	USER_FOLLOWS_KEY = "user-follows:%s:%d:%d" // <userID>:<limit>:<offset>
	IS_SUBSCRIBED_KEY = "%s-is-followed-on:%s" // <followerID>:<userID>
	PREPARE_USERNAME_KEY = "%s-prepare-for-registration" // <username>
	PREPARE_USER_EMAIL_KEY = "%s-prepare-for-registration" // <email>
)

func UserKey(userID string) string {
	return fmt.Sprintf(USER_KEY, userID)
}

func UserByUsernameKey(username string) string {
	return fmt.Sprintf(USER_BY_USERNAME_KEY, username)
}

func TempRegistrationCodeKey(code int) string {
	return fmt.Sprintf(TEMP_REGISTRATION_CODE_KEY, code)
}

func TempSignInCodeKey(code int) string {
	return fmt.Sprintf(TEMP_SIGNIN_CODE_KEY, code)
}

func SearchResultsKey(word string, limit int, offset int) string {
	return fmt.Sprintf(SEARCH_RESULTS_KEY, word, limit, offset)
}

func UserFollowersKey(userID string, limit int, offset int) string {
	return fmt.Sprintf(USER_FOLLOWERS_KEY, userID, limit, offset)
}

func UserFollowsKey(userID string, limit int, offset int) string {
	return fmt.Sprintf(USER_FOLLOWS_KEY, userID, limit, offset)
}

func IsSubscribedKey(followerID string, userID string) string {
	return fmt.Sprintf(IS_SUBSCRIBED_KEY, followerID, userID)
}

func PrepareUsernameKey(username string) string {
	return fmt.Sprintf(PREPARE_USERNAME_KEY, username)
}

func PrepareUserEmailKey(email string) string {
	return fmt.Sprintf(PREPARE_USER_EMAIL_KEY, email)
}
