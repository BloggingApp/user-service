package redisrepo

import "fmt"

const (
	USER_KEY           = "user:%s" // <userID>
	TEMP_REGISTRATION_CODE_KEY = "registration-code:%d" // <registration code>
	TEMP_SIGNIN_CODE_KEY = "sign-in-code:%d" // <sign-in code>
)

func UserKey(userID string) string {
	return fmt.Sprintf(USER_KEY, userID)
}

func TempRegistrationCodeKey(code int) string {
	return fmt.Sprintf(TEMP_REGISTRATION_CODE_KEY, code)
}

func TempSignInCodeKey(code int) string {
	return fmt.Sprintf(TEMP_SIGNIN_CODE_KEY, code)
}
