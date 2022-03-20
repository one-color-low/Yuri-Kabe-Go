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
2. `docker run --name yk-ap-ctn -d -p 6000:6000 -it yk-ap /bin/bash`
3. postmanでlocalhostへrequestを送る

## デバッグ
`docker exec -it yk-ap-ctn bash`