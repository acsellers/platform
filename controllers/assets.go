package controllers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/acsellers/platform/router"
)

type AssetModule struct {
	AssetLocation string
	MaxAge        time.Duration
}

func (am AssetModule) Load(sr *router.SubRoute) {
	files, err := ioutil.ReadDir(am.AssetLocation)
	if err != nil {
		return
	}
	for _, file := range files {
		if file.IsDir() {
			sr.Many(AssetController{
				BaseController: &router.BaseController{},
				Location:       file.Name(),
				CtrlPath:       filepath.Join(am.AssetLocation, file.Name()),
				MaxAge:         am.MaxAge,
			})
		} else {
			fmt.Println(file.Name())
		}
	}
}

type AssetController struct {
	*router.BaseController
	Location string
	CtrlPath string
	MaxAge   time.Duration
}

func (ac AssetController) Path() string {
	return ac.Location
}

func (ac AssetController) Show() router.Result {
	fn := filepath.Join(ac.CtrlPath, ac.Params[":"+ac.Location+"id"])
	_, err := os.Stat(fn)
	if err != nil {
		return router.NotFound{}
	}
	if ac.MaxAge.Seconds() != 0.0 {
		ac.Out.Header().Set(
			"Cache-Control",
			fmt.Sprintf("max-age=%.f", ac.MaxAge.Seconds()),
		)
	}
	http.ServeFile(ac.Out, ac.Request, fn)
	return router.NothingResult{}
}
