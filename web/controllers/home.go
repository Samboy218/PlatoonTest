package controllers

import (
    "net/http"
)

func (app *Application) HomeHandler(w http.ResponseWriter, r *http.Request) {
    data := &struct {
        QueryRet string
        Success bool
        Response bool
        User string
    }{
        QueryRet: "",
        Success: false,
        Response: false,
        User: app.Fabric.UserName,
    }

    if r.FormValue("submitted") == "true" {
        platID := r.FormValue("platID")
        payload, err := app.Fabric.QueryVal(platID)
        if err != nil {
            http.Error(w, "unable to invoke query with arg {" + platID +"}", 500)
            return
        }
       
        data.QueryRet = payload
        data.Success = true
        data.Response = true
    }
    renderTemplate(w, r, "home.html", data)
}
