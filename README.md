# goweber

#### 介绍
goweber是一個GO編寫的WEB框架，主要用於API服務。
支持功能：
- 路由
- 日志服務
- 配置服务
- 基於IP的限流
- 中間件服務，支持全局和路由級


#### 安装教程

```shell
go get github.com/sonkwl/goweber
```

#### 使用说明

```go
package goweber

import (
    "fmt"
    "net/http"
    "github.com/sonkwl/goweber"
)
func main() { 
    app := goweber.New()
    app.Get("/", func(w http.ResponseWriter, r *http.Request) { 
        fmt.Fprintf(w, "Hello World!")
    })
    app.Run()
}

```

#### 配置文件
請保證config.ini在執行文件同目錄下
```ini
[server]
# 網站端口
port = 8080

#  網絡日誌 logfile:目錄，logmax:文件最大大小
logfile = access.log
logmax = 1024000000

# 緩存器，單位Mb
cache = 1

# 限流 ipmax監控最大IP數量>0，開啓限流，ratelimit每秒訪問次數
ipmax=1000
ratelimit=5
```