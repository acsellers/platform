package controllers

import "github.com/acsellers/platform/router"

type ResetController struct{}

// SingleCtrl & MultiCtrl
func (r ResetController) Show() router.Result {
	return router.NotFound{}
}
func (r ResetController) Edit() router.Result {
	return router.NotFound{}
}
func (r ResetController) Update() router.Result {
	return router.NotFound{}
}
func (r ResetController) Delete() router.Result {
	return router.NotFound{}
}

// MultiCtrl only
func (r ResetController) New() router.Result {
	return router.NotFound{}
}
func (r ResetController) Create() router.Result {
	return router.NotFound{}
}
func (r ResetController) Index() router.Result {
	return router.NotFound{}
}
func (r ResetController) OtherBase(*router.SubRoute) {
}
func (r ResetController) OtherItem(*router.SubRoute) {
}
