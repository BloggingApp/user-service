package redisrepo

import "fmt"

const (
	USER_KEY           = "user:%s" // <userID>
	TEMP_USER_CODE_KEY = "%d"      // <activation code>
)

func UserKey(userID string) string {
	return fmt.Sprintf(USER_KEY, userID)
}

func TempUserCodeKey(code int) string {
	return fmt.Sprintf(TEMP_USER_CODE_KEY, code)
}
