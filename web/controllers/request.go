package controllers

import (
    "net/http"
)

func (app *Application) RequestHandler(w http.ResponseWriter, r *http.Request) {
    data := &struct {
        TransactionID string
        Success bool
        Response bool
    }{
        TransactionID: "",
        Success: false,
        Response: false,
    }
    if r.FormValue("submitted") == "true" {
        funcName := r.FormValue("func")
        platID := r.FormValue("platID")
        carID := r.FormValue("carID")
        txid, err := app.Fabric.Invoke(funcName, platID, carID)
        if err != nil {
            http.Error(w, "unable to invoke " + funcName + " with args {" + platID + ", " + carID + "}", 500)
            return
        }
        data.TransactionID = txid
        data.Success = true
        data.Response = true
    }
    renderTemplate(w, r, "request.html", data)
}
