package web

import (
    "fmt"
    "PlatoonTest/web/controllers"
    "net/http"
)

func Serve(app *controllers.Application, portNum int) {
    serv := http.NewServeMux()
    fs := http.FileServer(http.Dir("web/assets"))
    serv.Handle("/assets/", http.StripPrefix("/assets/", fs))

    serv.HandleFunc("/home.html", app.HomeHandler)
    serv.HandleFunc("/request.html", app.RequestHandler)

    serv.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, "/home.html", http.StatusTemporaryRedirect)
    })


    port := fmt.Sprintf(":%d", portNum)
    fmt.Println("listening (http://localhost"+ port +"/) ...")
    go func () {
        http.ListenAndServe(port, serv)
    }()
}
