URL-Shortener
===

## 簡介

這是基於 Dcard 2022 Backend Intern 的 [作業](https://drive.google.com/file/d/1AreBiHDUYXH6MI5OqWpKP-f6-W0zA8np/view?usp=sharing)，使用者可以自行發送 request 並且獲得一個獨特的 uid，之後可以利用此 uid 獲得縮網址前的網址內容

## 使用工具

- 資料庫
    - sqlite3
        - 為了能讓開發者能夠更快建立縮網址服務，所以使用 sqlite3 來讓架設速度增快
        - 因為可以建立許多 table，可以增加更多的開發空間
- 第三方函式庫
    - github.com/mattn/go-sqlite3
        - 為了能夠方便操作 sqlite3 而載入

## 功能

- 初始化
    - 檢查是否有 db 檔案，如未發現，將自動生成並建立相關 table
- 縮網址
    - 使用 `post` request 發出，並附上一個 json data
    - url 與 expireAt 檢查，如果無法符合要求將會回傳 `error 403`
        - url regex: `^http[s]?:\/\/(www\.)?(.*)?\/?(.)*?$`
        - expireAt regex: `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$`
        - expireAt 額外檢查是否已經過期，如為過去時間亦會回傳 `error 403`
- 解壓縮網址
    - 利用 get request 並搭配 `status code 302` 進行跳轉
    - 如果發現已經過期，將回傳 `error 404`
- 其他
    - 時間轉換以符合 db 中存成 `UTC +0` 的格式


## 測試

### 安裝函式庫

```bash=
go get -u github.com/mattn/go-sqlite3
```

### 開啟伺服器

```bash=
go run main.go
```

### 縮網址

Request:
```bash=
curl -v -X POST -H "Content-Type:application/json" http://127.0.0.1:4000/api/createUrl -d '{
    "url": "https://www.google.com",
    "expireAt": "2022-03-14T11:16:41Z"
}'
```

Response:
```bash=
{
    "id": "{shortId}",
    "message": "OK",
    "shortUrl": "http://localhost:4000/{shortId}"
}
```

### 解壓縮網址

Request:
```bash=
curl -L -X GET http://127.0.0.1:4000/{shortId}
```

Response:
```REDIRECT to original URL```
