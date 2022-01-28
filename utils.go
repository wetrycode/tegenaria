package tegenaria

import "github.com/google/uuid"

func GetUUID() string {
	u4 := uuid.New()
	uuid := u4.String()
	return uuid

}
