// Sample run-helloworld is a minimal Cloud Run service.
package main

import (
    "log"
    "net/http"
    "os"
    "html/template"
    "regexp"
    "path/filepath"
)

var templates = template.Must(template.ParseFiles("templates/edit.html", "templates/view.html", "templates/index.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
    entries, err := os.ReadDir("posts")
    if err != nil {
        http.NotFound(w, r)
        return
    }

    err = templates.ExecuteTemplate(w, "index.html", entries)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        log.Println("Error executing index template: ", err)
    }
}

var blogPath = regexp.MustCompile("^/blog/(.*)$")

func viewHandler(w http.ResponseWriter, r *http.Request) {
    m := blogPath.FindStringSubmatch(r.URL.Path)
    if m == nil {
        http.NotFound(w, r)
        return
    }

    log.Println(m[1])

    file, err := os.ReadFile(filepath.Join("posts", m[1]))
    if err != nil {
        http.NotFound(w, r)
        return
    }

    err = templates.ExecuteTemplate(w, "view.html", template.HTML(file))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        log.Println("Error executing view template: ", err)
    }
}

func main() {
    http.HandleFunc("/", indexHandler)
    http.HandleFunc("/blog/", viewHandler)

    assets := http.FileServer(http.Dir("assets"))
    http.Handle("/assets/", http.StripPrefix("/assets/", assets))

    fun := http.FileServer(http.Dir("fun"))
    http.Handle("/fun/", http.StripPrefix("/fun/", fun))

    // Determine port for HTTP service.
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
        log.Printf("defaulting to port %s", port)
    }

    // Start HTTP server.
    log.Printf("listening on port %s", port)
    if err := http.ListenAndServe(":" + port, nil); err != nil {
        log.Fatal(err)
    }
}
