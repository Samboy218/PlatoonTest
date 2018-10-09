package web

import (
    "fmt"
    "PlatoonTest/web/controllers"
    "net/http"
)

func Serve(app *controllers.Application) {
    fs := http.FileServer(http.Dir("web/assets"))
    http.Handle("/assets/", http.StripPrefix("/assets/", fs))

    http.HandleFunc("/home.html", app.HomeHandler)
    http.HandleFunc("/request.html", app.RequestHandler)

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        http.Redirect(w, r, "/home.html", http.StatusTemporaryRedirect)
    })


    fmt.Println("listening (http://localhost:8008/) ...")
    http.ListenAndServe(":8008", nil)
}
