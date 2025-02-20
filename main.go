// Sample run-helloworld is a minimal Cloud Run service.
package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "database/sql"
    "context"
    "net"
    "html/template"
    "regexp"
    "strconv"

    "cloud.google.com/go/cloudsqlconn"
    "github.com/go-sql-driver/mysql"
)

var (
    db *sql.DB
)

func connectWithConnector() {
    mustGetenv := func(k string) string {
        v := os.Getenv(k)
        if v == "" {
            log.Fatalf("Fatal Error in connect_connector.go: %s environment variable not set.", k)
        }
        return v
    }
    // Note: Saving credentials in environment variables is convenient, but not
    // secure - consider a more secure solution such as
    // Cloud Secret Manager (https://cloud.google.com/secret-manager) to help
    // keep passwords and other secrets safe.
    var (
        dbUser                 = mustGetenv("DB_USER")                  // e.g. 'my-db-user'
        dbPwd                  = mustGetenv("DB_PASS")                  // e.g. 'my-db-password'
        dbName                 = mustGetenv("DB_NAME")                  // e.g. 'my-database'
        instanceConnectionName = mustGetenv("INSTANCE_CONNECTION_NAME") // e.g. 'project:region:instance'
        usePrivate             = os.Getenv("PRIVATE_IP")
    )

    // WithLazyRefresh() Option is used to perform refresh
    // when needed, rather than on a scheduled interval.
    // This is recommended for serverless environments to
    // avoid background refreshes from throttling CPU.
    d, err := cloudsqlconn.NewDialer(context.Background(), cloudsqlconn.WithLazyRefresh())
    if err != nil {
        panic(fmt.Errorf("cloudsqlconn.NewDialer: %w", err))
    }
    var opts []cloudsqlconn.DialOption
    if usePrivate != "" {
        opts = append(opts, cloudsqlconn.WithPrivateIP())
    }
    mysql.RegisterDialContext("cloudsqlconn",
        func(ctx context.Context, addr string) (net.Conn, error) {
            return d.Dial(ctx, instanceConnectionName, opts...)
        })

    dbURI := fmt.Sprintf("%s:%s@cloudsqlconn(localhost:3306)/%s?parseTime=true",
        dbUser, dbPwd, dbName)

    dbPool, err := sql.Open("mysql", dbURI)
    if err != nil {
        panic(fmt.Errorf("sql.Open: %w", err))
    }

    db = dbPool
}

type Page struct {
    Id int
    Text string
}

func (p *Page) save() error {
    _, err := db.Exec("REPLACE INTO Articles (ArticleId, ArticleText) VALUES (?, ?)", strconv.Itoa(p.Id), p.Text)
    return err
}

func loadPage(id int) (*Page, error) {
    row := db.QueryRow("SELECT ArticleText FROM Articles WHERE ArticleId = ?;", id)
    var (
        text string
    )
    if err := row.Scan(&text); err != nil {
        return nil, err
    }
    return &Page{Id: id, Text: text}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, id int) {
    p, err := loadPage(id)
    if err != nil {
        return
    }
    renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, id int) {
    p, err := loadPage(id)
    if err != nil {
        p = &Page{Id: id}
    }
    renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, id int) {
    p := &Page{Id: id, Text: r.FormValue("body")}
    err := p.save()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/view/" + strconv.Itoa(p.Id), http.StatusFound)
}

var templates = template.Must(template.ParseFiles("edit.html", "view.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
    err := templates.ExecuteTemplate(w, tmpl + ".html", p)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, int)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        m := validPath.FindStringSubmatch(r.URL.Path)
        if m == nil {
            http.NotFound(w, r)
            return
        }
        i, err := strconv.Atoi(m[2])
        if err != nil {
            // ... handle error
            panic(err)
        }
        fn(w, r, i)
    }
}

func main() {
    connectWithConnector()

    http.HandleFunc("/view/", makeHandler(viewHandler))
    http.HandleFunc("/edit/", makeHandler(editHandler))
    http.HandleFunc("/save/", makeHandler(saveHandler))

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
