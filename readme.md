# Description
Yuri-Kabe-Go

# API
- `register`
- `upload`
- `get_list`

# How to Run
```
go run main.go
```


# Docker Test

## コンテナ単体
1. `docker build -t yk-ap . `
2. `docker run -it -d -p 6000:6000 yk-ap`
3. postmanでlocalhostへrequestを送る

