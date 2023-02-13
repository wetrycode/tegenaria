// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/wetrycode/tegenaria"
)

var apiLog *logrus.Entry = tegenaria.GetLogger("api") // engineLog engine runtime logger

type TegenariaAPI struct {
	G    *gin.Engine
	E    *tegenaria.CrawlEngine
	lock sync.Mutex
}
type statusResp struct {
	Request       uint64 `json:"request"`
	DownloadedErr uint64 `json:"downloaded_err"`
	Items         uint64 `json:"item"`
	Errors        uint64 `json:"errors"`
	Status        string `json:"status"`
	EngineId      string `json:"engine_id"`
}
type startRequest struct {
	Spider   string `json:"spider" binding:"required"`
	Interval int    `json:"interval" validate:"default=0"`
	IsMaster bool   `json:"is_master" validate:"default=false"`
}

func (t *TegenariaAPI) start(ctx *gin.Context) {
	t.lock.Lock()
	defer func() {
		t.lock.Unlock()
	}()
	var r startRequest
	appG := Gin{Ctx: ctx}
	err := ctx.ShouldBindJSON(&r)
	if err != nil {
		apiLog.Errorf("start spider params error:%s", err.Error())
		appG.Response(http.StatusBadRequest, http.StatusBadRequest, nil)
		return
	}
	status := t.E.GetStatusOn()
	if status == tegenaria.ON_START || status == tegenaria.ON_PAUSE {
		appG.Response(http.StatusOK, SPIER_REPEAT_START, "")
		return

	} else {
		if r.Interval <= 0 {
			go t.E.Start(r.Spider)
		} else {
			t.E.SetInterval(time.Duration(r.Interval * int(time.Second)))
			go t.E.StartWithTicker(r.Spider)
		}
		time.Sleep(1 * time.Second)
	}
	appG.Response(http.StatusOK, 200, "")

}

func (t *TegenariaAPI) stop(ctx *gin.Context) {
	t.lock.Lock()
	defer func() {
		t.lock.Unlock()
	}()
	t.E.SetStatus(tegenaria.ON_STOP)
	t.E.StopTicker()
	appG := Gin{Ctx: ctx}

	appG.Response(http.StatusOK, 200, "")

}

func (t *TegenariaAPI) pause(ctx *gin.Context) {
	t.lock.Lock()
	defer func() {
		t.lock.Unlock()
	}()
	if t.E.GetStatusOn() == tegenaria.ON_START {
		t.E.SetStatus(tegenaria.ON_PAUSE)
	}
	appG := Gin{Ctx: ctx}

	appG.Response(http.StatusOK, 200, "")
}

func (t *TegenariaAPI) status(ctx *gin.Context) {
	statistic := t.E.GetStatic()
	ip, _ := tegenaria.GetMachineIp()
	rsp := statusResp{
		Request:       statistic.GetRequestSent(),
		DownloadedErr: statistic.GetDownloadFail(),
		Items:         statistic.GetItemScraped(),
		Errors:        statistic.GetErrorCount(),
		Status:        t.E.GetStatusOn().GetTypeName(),
		EngineId:      fmt.Sprintf("%s:%s", ip, tegenaria.GetEngineId()),
	}
	appG := Gin{Ctx: ctx}

	appG.Response(http.StatusOK, 200, rsp)

}
func (t *TegenariaAPI) Server() {
	server := &http.Server{
		Addr:         "0.0.0.0:12138",
		Handler:      t.G,
		ReadTimeout:  time.Duration(10 * time.Second),
		WriteTimeout: time.Duration(10 * time.Second),
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
func NewAPI(engine *tegenaria.CrawlEngine) *TegenariaAPI {
	API := &TegenariaAPI{
		E: engine,
	}
	g := SetUp()

	v1Router := g.Group("/api/v1")
	v1Router.POST("/start", API.start)
	v1Router.POST("/stop", API.stop)
	v1Router.POST("/pause", API.pause)
	v1Router.GET("/status", API.status)
	API.G = g
	return API

}
