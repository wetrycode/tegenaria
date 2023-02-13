/**
 * Copyright (c) 2023 wetrycode
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package api

import (
	"github.com/gin-gonic/gin"
)

type Gin struct {
	Ctx *gin.Context
}

func SetUp() *gin.Engine {
	engine := gin.New()
	return engine
}
