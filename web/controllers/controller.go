package controllers

import (
    "fmt"
    "PlatoonTest/blockchain"
    "html/template"
    "net/http"
    "os"
    "path/filepath"
)

type Application struct {
    Fabric *blockchain.ClientSetup
}

func renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data interface{}) {
    lp := filepath.Join("web", "templates", "layout.html")
    tp := filepath.Join("web", "templates", templateName)

    info, err := os.Stat(tp)
    if err != nil {
        if os.IsNotExist(err) {
            http.NotFound(w, r)
            return
        }
    }

    if info.IsDir() {
        http.NotFound(w, r)
        return
    }

    resultTemplate, err := template.ParseFiles(tp, lp)
    if err != nil {
        fmt.Println(err.Error())
        http.Error(w, http.StatusText(500), 500)
        return
    }
    if err := resultTemplate.ExecuteTemplate(w, "layout", data); err != nil {
        fmt.Println(err.Error())
        http.Error(w, http.StatusText(500), 500)
        return
    }
}
