package controllers

import (
    "net/http"
    "encoding/json"
    "fmt"
)

type platoonUser struct {
    ID          string
    CurrPlat    string
    Reputation  int
    Money       int
    LastMove    int64
}
type platoon struct {
    ID string
    CurrSpeed int
    //timestamp of last change
    LastMove int64
    //distance (in miles) since the leaer was last payed
    Distance int
    Members []string
}


func (app *Application) HomeHandler(w http.ResponseWriter, r *http.Request) {
    data := &struct {
        QueryRet platoon
        Success bool
        Response bool
        User string
    }{
        QueryRet: platoon{},
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

        if platID == "platoons" {
            var platoonIDs []string
            err = json.Unmarshal([]byte(payload), &platoonIDs)
            if err != nil {
                http.Error(w, fmt.Sprintf("unable to decode JSON response: %v", err), 500)
                return
            }
            var platoons []platoon
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
                platoons = append(platoons, tempPlat)
                tempPlat = platoon{}
            }
            dataPlat := &struct {
                QueryRet []platoon
                Success bool
                Response bool
                User string
            }{
                QueryRet: platoons,
                Success: true,
                Response: true,
                User: app.Fabric.UserName,
            }
            renderTemplate(w, r, "home-platoons.html", dataPlat)
            return
        }
        if platID == "users" {
            var users  []platoonUser
            err = json.Unmarshal([]byte(payload), &users)
            if err != nil {
                http.Error(w, fmt.Sprintf("unable to decode JSON response: %v", err), 500)
                return
            }
            dataUser := &struct {
                QueryRet []platoonUser
                Success bool
                Response bool
                User string
            }{
                QueryRet: users,
                Success: true,
                Response: true,
                User: app.Fabric.UserName,
            }
            renderTemplate(w, r, "home-users.html", dataUser)
            return
        }

        //they are requesting a single platoon
        var plat platoon
        err = json.Unmarshal([]byte(payload), &plat)
        if err != nil {
            http.Error(w, fmt.Sprintf("error decoding JSON response: %v", err), 500)
            return
        }
        data.QueryRet = plat
        data.Success = true
        data.Response = true
    }
    renderTemplate(w, r, "home.html", data)
}
