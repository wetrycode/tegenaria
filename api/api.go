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
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/wetrycode/tegenaria"
)

var apiLog *logrus.Entry = tegenaria.GetLogger("api") // engineLog engine runtime logger

type TegenariaAPI struct {
	G           *gin.Engine
	E           *tegenaria.CrawlEngine
	lock        sync.Mutex
	AdminServer string
	ticker      *time.Ticker
	server      string
	host        string
}
type statusResp struct {
	Request  uint64   `json:"requests"`
	Items    uint64   `json:"items"`
	Errors   uint64   `json:"errors"`
	Status   string   `json:"status"`
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Host     string   `json:"host"`
	Duration float64  `json:"duration"`
	StartAt  string   `json:"start"`
	Role     string   `json:"role"`
	Spiders  []string `json:"spiders"`
}
type startRequest struct {
	Spider   string `json:"name" binding:"required"`
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
		t.E.SetStatus(tegenaria.ON_STOP)
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
	t.E.SetStatus(tegenaria.ON_PAUSE)
	apiLog.Infof("暂停爬虫")
	appG := Gin{Ctx: ctx}

	appG.Response(http.StatusOK, 200, "")
}
func (t *TegenariaAPI) getStartAndDuration() (string, float64) {
	startAt := t.E.GetStartAt()
	duration := t.E.GetDuration()
	if startAt == 0 {
		return "", duration
	}
	timeStr := time.Unix(startAt, 0).Format("2006-01-02 15:04:05")
	return timeStr, duration
}
func (t *TegenariaAPI) getStatus() *statusResp {
	statistic := t.E.GetStatic()
	startAt, duration := t.getStartAndDuration()
	spider := t.E.GetCurrentSpider()
	name := ""
	if spider != nil {
		name = spider.GetName()
	}
	rsp := &statusResp{
		Request:  statistic.Get(tegenaria.RequestStats),
		Items:    statistic.Get(tegenaria.ItemsStats),
		Errors:   statistic.Get(tegenaria.ErrorStats),
		Status:   t.E.GetStatusOn().GetTypeName(),
		ID:       tegenaria.GetEngineID(),
		Host:     t.host,
		StartAt:  startAt,
		Duration: duration,
		Name:     name,
		Role:     t.E.GetRole(),
		Spiders:  t.E.GetSpiders().GetAllSpidersName(),
	}
	return rsp
}
func (t *TegenariaAPI) status(ctx *gin.Context) {

	appG := Gin{Ctx: ctx}
	rsp := t.getStatus()
	appG.Response(http.StatusOK, 200, rsp)

}
func (t *TegenariaAPI) GetAllSpiders(ctx *gin.Context) {
	appG := Gin{Ctx: ctx}
	spiders := t.E.GetSpiders()
	names := spiders.GetAllSpidersName()
	appG.Response(http.StatusOK, 200, names)
}
func (t *TegenariaAPI) Server(host string, port int) {
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      t.G,
		ReadTimeout:  time.Duration(10 * time.Second),
		WriteTimeout: time.Duration(10 * time.Second),
	}
	ip, _ := tegenaria.GetMachineIP()

	t.server = fmt.Sprintf("%s:%d", host, port)
	t.host = fmt.Sprintf("%s:%d", ip, port)
	go t.heartbeat()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
func (t *TegenariaAPI) heartbeat() {
	t.ticker = time.NewTicker(3 * time.Second)
	for range t.ticker.C {
		status := t.getStatus()
		api, err := url.JoinPath(t.AdminServer, "/api/v1/update")
		if err != nil {
			apiLog.Errorf("心跳包发送失败:%s", err.Error())
			panic(err)
		}
		jsonBytes, err := json.Marshal(status)
		if err != nil {
			apiLog.Errorf("心跳包发送失败:%s", err.Error())

		}
		rsp, err := http.Post(api, "application/json; charset=UTF-8", bytes.NewReader(jsonBytes))
		if err != nil {
			apiLog.Errorf("心跳包发送失败:%s", err.Error())

		} else {
			if rsp.StatusCode != 200 {
				apiLog.Errorf("心跳包发送失败,状态码为:%d", rsp.StatusCode)

			}
		}
	}
}
func NewAPI(engine *tegenaria.CrawlEngine, adminServer string) *TegenariaAPI {
	API := &TegenariaAPI{
		E:           engine,
		AdminServer: adminServer,
		ticker:      nil,
	}
	g := SetUp()

	v1Router := g.Group("/api/v1")
	v1Router.POST("/start", API.start)
	v1Router.POST("/stop", API.stop)
	v1Router.POST("/pause", API.pause)
	v1Router.GET("/status", API.status)
	v1Router.GET("/spiders", API.GetAllSpiders)

	API.G = g
	return API

}
