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

    "cloud.google.com/go/cloudsqlconn"
    "github.com/go-sql-driver/mysql"
)

func connectWithConnector() (*sql.DB, error) {
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
        return nil, fmt.Errorf("cloudsqlconn.NewDialer: %w", err)
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
        return nil, fmt.Errorf("sql.Open: %w", err)
    }
    return dbPool, nil
}

func main() {
    db, err := connectWithConnector()
    if err != nil {
        panic(err)
    }

    log.Print("starting server...")

    articleHandlerWrapper := func(w http.ResponseWriter, r *http.Request) {
        log.Printf("hello world")
        row := db.QueryRow("SELECT * FROM Articles WHERE ArticleId = 1;")
        var (
            id int64
            text string
        )
        if err := row.Scan(&id, &text); err != nil {
            log.Fatal(err)
        }

        fmt.Fprintf(w, "id %d text is %s\n", id, text)
        fmt.Fprintf(w, "help")
    }
    http.HandleFunc("/", articleHandlerWrapper)

    // Determine port for HTTP service.
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
        log.Printf("defaulting to port %s", port)
    }

    // Start HTTP server.
    log.Printf("listening on port %s", port)
    if err := http.ListenAndServe(":"+port, nil); err != nil {
        log.Fatal(err)
    }
}
