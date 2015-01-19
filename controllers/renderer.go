package controllers

import (
	"bytes"

	"github.com/acsellers/multitemplate"
	"github.com/acsellers/platform/router"
)

type RenderableCtrl struct {
	*router.BaseController
	Template     *multitemplate.Template
	Page, Layout string
}

func NewRenderableCtrl(tmpl *multitemplate.Template) RenderableCtrl {
	return RenderableCtrl{
		&router.BaseController{},
		tmpl,
		"", "",
	}
}

func (rc RenderableCtrl) Render() router.Result {
	ctx := &multitemplate.Context{
		Main:   rc.Page,
		Layout: rc.Layout,
		Dot:    rc.Context,
	}
	buf := &bytes.Buffer{}
	err := rc.Template.ExecuteContext(buf, ctx)
	if err != nil {
		return router.InternalError{err}
	} else {
		return router.Rendered{Content: buf}
	}
}
