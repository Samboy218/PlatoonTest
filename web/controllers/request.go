package controllers

import (
    "net/http"
)

func (app *Application) RequestHandler(w http.ResponseWriter, r *http.Request) {
    data := &struct {
        TransactionID string
        Success bool
        Response bool
        User string
    }{
        TransactionID: "",
        Success: false,
        Response: false,
        User: app.Fabric.UserName,
    }
    if r.FormValue("submitted") == "true" {
        funcName := r.FormValue("func")
        platID := r.FormValue("platID")
        txid, err := app.Fabric.Invoke(funcName, platID, "")
        if err != nil {
            if txid == "timeout" {
            //http.Error(w, "unable to invoke " + funcName + " with args {" + platID + "}" + err.Error(), 500)
                data.TransactionID = "Did not receive chaincode event, but the transaction probably still went through"
            }else {
                data.TransactionID = err.Error()
            }
            data.Response = true
            data.Success = false
        }else {
            data.TransactionID = txid
            data.Success = true
            data.Response = true
        }
    }
    renderTemplate(w, r, "request.html", data)
}
