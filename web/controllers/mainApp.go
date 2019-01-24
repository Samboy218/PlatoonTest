package controllers

import (
    "net/http"
    "encoding/json"
    "fmt"
    "strings"
)

func (app *Application) MainAppHandler(w http.ResponseWriter, r *http.Request) {
    data := &struct {
        Platoons []platoonDeep
        Users []platoonUser
        CurrUser platoonUser
        CurrPlat platoonDeep
        Success bool
        Response bool
        TxSuccess bool
        TxResponse bool
        TransactionID string
    }{
        Platoons: make([]platoonDeep, 0),
        Users: make([]platoonUser, 0),
        CurrUser: platoonUser{ID: app.Fabric.UserName},
        CurrPlat: platoonDeep{},
        Success: false,
        Response: false,
        TxSuccess: false,
        TxResponse: false,
        TransactionID: "",
    }

    if r.FormValue("submitted") == "true" {
        funcName := r.FormValue("func")
        platID := r.FormValue("platID")
        txid, err := app.Fabric.Invoke(funcName, platID, "")
        if err != nil {
            if txid == "timeout" {
                data.TransactionID = "Did not receive chaincode event (timeout)."
            }else {
                data.TransactionID = err.Error()
            }
            data.TxResponse = true
            data.TxSuccess = false
        }else {
            data.TransactionID = txid
            data.TxSuccess = true
            data.TxResponse = true
        }

    }

    //Get all platoons and Users
    var users  []platoonUser
    payload, err := app.Fabric.QueryVal("users")
    if payload != "" {
        err = json.Unmarshal([]byte(payload), &users)
        if err != nil {
            http.Error(w, fmt.Sprintf("unable to decode JSON response: %v", err), 500)
            return
        }
    }
    

    var platoonsList []platoon
    var platoonIDs []string
    payload, err = app.Fabric.QueryVal("platoons")
    if payload != "" {
        err = json.Unmarshal([]byte(payload), &platoonIDs)
        if err != nil {
            http.Error(w, fmt.Sprintf("unable to decode JSON response: %v", err), 500)
            return
        }
        var tempPlat platoon
        for _, id := range platoonIDs {
            payload, err = app.Fabric.QueryVal(id)
            if err != nil {
                http.Error(w, fmt.Sprintf("unable to get platoon {%s}: %v", id, err), 500)
                return
            }
            if payload != "" {
                err = json.Unmarshal([]byte(payload), &tempPlat)
                if err != nil {
                    http.Error(w, fmt.Sprintf("unable to decode JSON: %v\n%s", err, payload), 500)
                    return
                }
            }
            platoonsList = append(platoonsList, tempPlat)
            tempPlat = platoon{}
        }
    }
    for i, user := range users {
        splitID := strings.Split(user.ID, "@")
        if splitID[0] == data.CurrUser.ID {
            data.CurrUser = user
            //now remove the user from the main user array
            users = append(users[:i], users[i+1:]...)
            break
        }
    }

    //distribute users into their platoons
    noPlat := make([]platoonUser, 0)
    platoons := make([]platoonDeep, 0)
    for _, currPlat := range platoonsList {
        var tempPlat platoonDeep
        tempPlat.ID = currPlat.ID
        tempPlat.LastMove = currPlat.LastMove
        tempPlat.Distance = currPlat.Distance
        tempPlat.CurrSpeed = currPlat.CurrSpeed
        for _, currUserID := range currPlat.Members {
            for _, currUser := range users {
                if currUser.ID == currUserID {
                    tempPlat.Members = append(tempPlat.Members, currUser)
                }
            }
            if data.CurrUser.ID == currUserID {
                tempPlat.Members = append(tempPlat.Members, data.CurrUser)
            }
        }
        if currPlat.ID == data.CurrUser.CurrPlat {
            data.CurrPlat = tempPlat
        }else {
            platoons = append(platoons, tempPlat)
        }
    }
    for _, currUser := range users {
        if currUser.CurrPlat == "" {
            noPlat = append(noPlat, currUser)
        }
    }

    data.Users = noPlat
    data.Platoons = platoons
    data.Success = true
    data.Response = true

    renderTemplate(w, r, "mainApp.html", data)
}
