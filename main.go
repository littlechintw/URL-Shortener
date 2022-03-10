package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "regexp"
    "database/sql"
    "time"
    "math/rand"

    _ "github.com/mattn/go-sqlite3"
)

var backendIp = "http://localhost"
var backendPort = "4000"
var backendUrl = backendIp + ":" + backendPort

// shortUrl api request json body
type CreateShortUrl struct {
    Url string `json:"url"`
    ExpireAt string `json:"expireAt"`
}

// get now time with the format of "2006-01-02T15:04:05Z"
func getNowTime() string {
    loc, _: = time.LoadLocation("UTC")
    return time.Now().In(loc).Format("2006-01-02T15:04:05Z")
}

// init sqlite
func sqliteInit() {
    db, err: = sql.Open("sqlite3", "./shortUrl.db")
    checkErr(err)

    sql_table: = `
    CREATE TABLE IF NOT EXISTS url(
        createTime DATE NULL,
        url VARCHAR(256) NULL,
        shortId VARCHAR(10) NULL,
        expireTime DATE NULL
    );
    `
    db.Exec(sql_table)
    db.Close()
}

// compare string whether is url
func urlFormatVerify(url string) bool {
    match, _: = regexp.MatchString(`^http[s]?:\/\/(www\.)?(.*)?\/?(.)*?$`, url)
    return match
}

// compare string whether is expireTime and not expired
func expireTimeVerify(expireTime string) bool {
    match, _: = regexp.MatchString(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`, expireTime)
    if match {
        nowTime: = getNowTime()
            // compare expireTime whether is expired
        if expireTime > nowTime {
            return true
        }
    }
    return false
}

// compare expireTime whether is expired
func isExpired(expireTime string) bool {
    nowTime: = getNowTime()
    if expireTime < nowTime {
        return true
    }
    return false
}

// verify shortUrl is valid and the shortUrl in the database is not expired
func shortUrlVerify(urlPath string)(bool, string) {
    // verify shortUrl is valid
    match, _: = regexp.MatchString(`^/[a-zA-Z0-9]{4}$`, urlPath)
    if !match {
        return false, ""
    }

    // verify shortId whether is in the database
    var requestShortId = urlPath[1: ]
    db, err: = sql.Open("sqlite3", "./shortUrl.db")
    checkErr(err)
    rows, err: = db.Query("SELECT * FROM url WHERE shortId = ?", requestShortId)
    checkErr(err)

    var createTime time.Time
    var url string
    var shortId string
    var expireTime time.Time

    defer rows.Close()
    for rows.Next() {
        err = rows.Scan( & createTime, & url, & shortId, & expireTime)
        checkErr(err)
        fmt.Println("Found " + shortId + " in database")
        fmt.Println("Url: " + url)
            // verify shortId whether is expired
        if !isExpired(expireTime.Format("2006-01-02T15:04:05Z")) {
            return true, url
        }
        return false, ""
    }
    return false, ""
}

// check error
func checkErr(err error) {
    if err != nil {
        panic(err)
    }
}

// check the shortId whether is in the database
func checkIdExist(id string) bool {
    db, err: = sql.Open("sqlite3", "./shortUrl.db")
    checkErr(err)
    rows, err: = db.Query("SELECT * FROM url WHERE shortId = ?", id)
    checkErr(err)
    defer rows.Close()
    for rows.Next() {
        return true
    }
    return false
}

// create shortId by random
func shortIdGenerater() string {
    var n int = 4 // length of shortId
    rand.Seed(time.Now().UnixNano())
    var letterRunes = [] rune("01234556789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
    b: = make([] rune, n)
    for i: = range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }

    for (checkIdExist(string(b))) {
        b: = make([] rune, n)
        for i: = range b {
            b[i] = letterRunes[rand.Intn(len(letterRunes))]
        }
    }
    return string(b)
}

// write a record into database
func writeShortUrl(createTime string, url string, shortId string, expireTime string) {
    db, err: = sql.Open("sqlite3", "./shortUrl.db")
    checkErr(err)
        // insert
    stmt, err: = db.Prepare("INSERT INTO url(createTime, url, shortId, expireTime) values(?,?,?,?)")
    checkErr(err)

    res, err: = stmt.Exec(createTime, url, shortId, expireTime)
    checkErr(err)

    id, err: = res.LastInsertId()
    checkErr(err)

    fmt.Println(id)
    db.Close()
}

// api for create shortId
func shortUrlCreate(w http.ResponseWriter, r * http.Request) {
    var data CreateShortUrl

    err: = json.NewDecoder(r.Body).Decode( & data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // set the expireTime to UTC +0
    layout: = "2006-01-02T15:04:05Z"
    tmp, err: = time.Parse(layout, data.ExpireAt)
    tmp = tmp.Add(time.Hour * -8)
    expireAt: = tmp.Format(layout)

    if err != nil {
        fmt.Println(err)
    }

    // if the url is not valid or the expireTime is not valid, will return 403 error
    if !(urlFormatVerify(data.Url) && expireTimeVerify(expireAt)) {
        w.WriteHeader(403)
        w.Header().Set("Content-Type", "application/json")
        resp: = make(map[string] string)
        resp["message"] = "Forbidden"
        jsonResp, err: = json.Marshal(resp)
        if err != nil {
            log.Fatalf("Error happened in JSON marshal. Err: %s", err)
        }
        w.Write(jsonResp)
        return
    }

    createTime: = getNowTime()
    shortId: = shortIdGenerater()
    writeShortUrl(createTime, data.Url, shortId, expireAt)

    // Return 200 OK and the shortUrl details.
    w.WriteHeader(200)
    w.Header().Set("Content-Type", "application/json")
    resp: = make(map[string] string)
    resp["message"] = "OK"
    resp["id"] = shortId
    resp["shortUrl"] = backendUrl + "/" + shortId
    jsonResp, err: = json.Marshal(resp)
    if err != nil {
        log.Fatalf("Error happened in JSON marshal. Err: %s", err)
    }
    w.Write(jsonResp)
}

// api for redirect shortId
func redirect(w http.ResponseWriter, r * http.Request) {
    verify, url: = shortUrlVerify(r.URL.Path)

    if verify {
        w.Header().Set("Location", url)
        w.WriteHeader(302)
    } else {
        w.WriteHeader(404)
        w.Header().Set("Content-Type", "application/json")
        resp: = make(map[string] string)
        resp["message"] = "Not Found"
        jsonResp, err: = json.Marshal(resp)
        if err != nil {
            log.Fatalf("Error happened in JSON marshal. Err: %s", err)
        }
        w.Write(jsonResp)
    }
}

func main() {
    // init the sqlite3 database
    sqliteInit()

    // start the server
    mux: = http.NewServeMux()
    mux.HandleFunc("/api/createUrl", shortUrlCreate)
    mux.HandleFunc("/", redirect)

    err: = http.ListenAndServe(":" + backendPort, mux)
    log.Fatal(err)
}