/**
 * Copyright (c) 2023 wetrycode
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package api

const (
	SUCCESS        = 200
	ERROR          = 500
	INVALID_PARAMS = 400
	NOT_FOUND      = 404

	APP_HEALTH_OK       = 1000
	MONGO_CONNECT_ERROR = 1001
	REDIS_CONNECT_ERROR = 1002
	SPIER_REPEAT_START  = 1003
)

var MsgFlags = map[int]string{
	SUCCESS:            "ok",
	ERROR:              "fail",
	INVALID_PARAMS:     "bad request",
	NOT_FOUND:          "resource not found",
	SPIER_REPEAT_START: "spider repeat start",
}

// GetMsg get error information based on Code
func GetMsg(code int) string {
	msg, ok := MsgFlags[code]
	if ok {
		return msg
	}

	return MsgFlags[ERROR]
}
