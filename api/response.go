/**
 * Copyright (c) 2023 wetrycode
 *
 * This software is released under the MIT License.
 * https://opensource.org/licenses/MIT
 */

package api

type Response struct {
	APIVersion string      `json:"api"`
	Code       int         `json:"code"`
	Message    string      `json:"msg"`
	Data       interface{} `json:"data"`
}

// Response 响应函数
func (g *Gin) Response(httpCode, errCode int, data interface{}) {
	g.Ctx.JSON(httpCode, Response{
		APIVersion: "v0.0.1",
		Code:       errCode,
		Message:    GetMsg(errCode),
		Data:       data,
	})
}
