package goweber

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"golang.org/x/time/rate"
	"sync"
)

// 中間件類型
// Middleware type
type MiddlewareFunc func(r *http.Request) error

// Apper 是应用程序的主结构体，包含路由映射、配置信息、端口、日志等信息
// Apper is the main struct of the application, containing route mappings, configuration information, port, logs, etc.
type Apper struct {
	// 路由映射表，第一层key为HTTP方法，第二层key为路径，值为处理函数
	// Route mapping table, first level key is HTTP method, second level key is path, value is handler function
	rmap map[string]map[string]http.HandlerFunc
	// 配置信息结构体指针
	// Configuration information struct pointer
	Config *Configer
	// 服务器监听端口
	// Server listening port
	port string
	// 用于传递日志消息的通道
	// Channel used to pass log messages
	msg chan string
	// 日志文件句柄
	// Log file handle
	logfile *os.File
	// 日志记录器
	// Logger
	log *log.Logger
	// 日志文件最大大小限制
	// Maximum log file size limit
	logmax int64
	// 全局中間件
	// Global middleware
	gMiddleware []MiddlewareFunc
	// 路由中間件
	// Route middleware
	rMiddleware map[string]map[string][]MiddlewareFunc
	// 限流器
	// Rate limiter
	iplimiter map[string]*rate.Limiter
	// 最大監控IP數量
	// Maximum monitored IP count
	ipmax int
	// 限流
	// Rate limit
	ratelimit int
	// 鎖
	// Lock
	mu sync.Mutex 
	// 用戶行爲監控
	// user behavior monitoring
	Bh *Behaver
}

// New 创建并初始化一个新的Apper实例
// New creates and initializes a new Apper instance
func New() *Apper {
	// * 读取配置文件
	// * Read configuration file
	app := &Apper{
		rmap:   make(map[string]map[string]http.HandlerFunc),
		port:   "8080",
		logmax: 102400000,
		Config: &Configer{
			params: make(map[string]map[string]string),
		},
		msg: make(chan string),
		ratelimit: 100,
		Bh: NewBehaver(),
	}
	app.SetConfig()
	app.SetLog()
	app.SetPort()
	app.SetLimit()
	app.SetBehaver()
	return app
}

// Close 关闭应用程序资源，包括日志文件和消息通道
// Close closes application resources, including log file and message channel
func (this *Apper) Close() {
	if this.logfile != nil {
		this.logfile.Close()
	}
	close(this.msg)
}

// SetConfig 从config.ini文件中读取配置信息
// SetConfig reads configuration information from config.ini file
func (this *Apper) SetConfig() {
	//* 讀取配置文件
	//* Read configuration file
	configfile, err := os.OpenFile("config.ini", os.O_RDONLY, 0666)
	if err != nil {
		// fmt.Println("讀取文件config.ini失敗")
		// fmt.Println("Failed to read config.ini file")
		panic("讀取文件config.ini失敗")
		// panic("Failed to read config.ini file")
	}
	// defer configfile.Close()
	this.Config.SetFile(configfile)
	// fmt.Println(this.Config.params)
}

// SetLog 根据配置设置日志记录器
// SetLog sets up the logger according to configuration
func (this *Apper) SetLog() {
	// * 检测是否有访问日志
	// * Check if there is access log
	logpath := this.Config.Get("server", "logfile")
	if logpath != "" {
		logfile, err := os.OpenFile(logpath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			defer this.logfile.Close()
			panic(err)
		}
		this.logfile = logfile
		this.log = log.New(logfile, "", log.LstdFlags)
	}
	logmax := this.Config.Get("server", "logmax")
	if logmax != "" {
		this.logmax, _ = strconv.ParseInt(logmax, 10, 64)
	}
}

// SetPort 从配置中设置服务器端口，默认为8080
// SetPort sets the server port from configuration, default is 8080
func (this *Apper) SetPort() {
	if this.Config.Get("server", "port") != "" {
		this.port = this.Config.Get("server", "port")
	}
}

// SetLimit 設置限流
// SetLimit set rate limiting
func (this *Apper) SetLimit() {
	if this.Config.Get("server", "ipmax") != "" {
		this.ipmax ,_= strconv.Atoi(this.Config.Get("server", "ipmax"))
		// 開啓限流
		// Enable rate limiting
		if this.ipmax>0 {
			this.iplimiter=make(map[string]*rate.Limiter,0)
		}
	}
	if this.Config.Get("server", "ratelimit") != "" {
		this.ratelimit,_= strconv.Atoi(this.Config.Get("server", "ratelimit"))
	}
}

// SetBehaver
func (this *Apper) SetBehaver() {
	if this.Config.Get("behaver", "ipmax") != "" {
		this.Bh.IpMax,_= strconv.Atoi(this.Config.Get("behaver", "ipmax"))
	} 
	if this.Config.Get("behaver", "ipmax") != "" {
		this.Bh.Expire,_= strconv.ParseInt(this.Config.Get("behaver", "expire"),10,64)
	}
	if this.Config.Get("behaver", "cleansecond") != "" {
		this.Bh.CleanSecond,_= strconv.ParseInt(this.Config.Get("behaver", "cleansecond"),10,64)
	} 
	if this.Bh.IpMax > 0 {
		go this.Bh.Clear()
	}
}

// GetClientIP 获取客户端真实IP地址
// GetClientIP get client real IP address
func (this *Apper) GetClientIP(r *http.Request) string {
	// 优先从X-Forwarded-For头获取IP
	// Prioritize getting IP from X-Forwarded-For header
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		// 如果X-Forwarded-For为空，则尝试从X-Real-IP头获取
		// If X-Forwarded-For is empty, try to get from X-Real-IP header
		ip = r.Header.Get("X-Real-IP")
	}
	// 如果两个头都没有值，则使用RemoteAddr
	// If both headers have no value, use RemoteAddr
	if ip == "" {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return "unknown" // 格式化错误，返回未知IP
			// Formatting error, return unknown IP
		}
		return ip
	}

	return ip
}

// 全局中間件處理
// Global middleware processing
func (this *Apper) Use(middleware ...MiddlewareFunc) {
	if this.gMiddleware == nil {
		this.gMiddleware = make([]MiddlewareFunc, 0)
	}
	this.gMiddleware = append(this.gMiddleware, middleware...)
}

// Get 注册GET请求的路由处理函数
// Get registers the route handler function for GET requests
func (this *Apper) Get(path string, f http.HandlerFunc, mid ...MiddlewareFunc) {
	this.Route("GET", path, f, mid...)
}

// Post 注册POST请求的路由处理函数
// Post registers the route handler function for POST requests
func (this *Apper) Post(path string, f http.HandlerFunc, mid ...MiddlewareFunc) {
	this.Route("POST", path, f, mid...)
}

// Route 注册指定HTTP方法的路由处理函数
// Route registers the route handler function for the specified HTTP method
func (this *Apper) Route(method string, path string, f http.HandlerFunc, mid ...MiddlewareFunc) {
	// 添加到路由中間件
	// Add to route middleware
	for _, m := range mid {
		if this.rMiddleware == nil {
			this.rMiddleware = make(map[string]map[string][]MiddlewareFunc)
		}
		if this.rMiddleware[method] == nil {
			this.rMiddleware[method] = make(map[string][]MiddlewareFunc)
		}
		if this.rMiddleware[method][path] == nil {
			this.rMiddleware[method][path] = make([]MiddlewareFunc, 0)
		}
		this.rMiddleware[method][path] = append(this.rMiddleware[method][path], m)
	}
	if this.rmap[method] == nil {
		this.rmap[method] = make(map[string]http.HandlerFunc)
	}
	this.rmap[method][path] = f
}

// Logger 处理日志记录，监听消息通道并将日志写入文件或控制台
// Logger handles log recording, listens to the message channel and writes logs to file or console
func (this *Apper) Logger() {
	// fmt.Println("Logger")
	for msg := range this.msg {
		if this.log != nil {
			info, _ := this.logfile.Stat()
			if info.Size() > this.logmax && this.logmax > 0 {
				//* 日志空間超過上綫，新增日志文件
				//* Log space exceeds limit, create new log file
				this.logfile.Close()
				infoname := info.Name()
				os.Rename(infoname, infoname+strconv.FormatInt(time.Now().Unix(), 10))
				this.logfile, _ = os.OpenFile(infoname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				this.log = log.New(this.logfile, "", log.LstdFlags)
			}
			this.log.Println(msg)
		} else {
			fmt.Println(msg)
		}
	}
}

// getLimiter 获取限流器
// getLimiter get rate limiter
func (this *Apper) getLimiter(ip string) *rate.Limiter {
	this.mu.Lock()
	defer this.mu.Unlock()
	
	if limiter,ok := this.iplimiter[ip]; ok {
		return limiter
	}
	// IP數量
	// IP count
	if len(this.iplimiter) >= this.ipmax {
		// 清除50%最早加入的IP
		// Clear 50% of the earliest added IPs
		this.clearLimiter()
	}
	this.iplimiter[ip] = rate.NewLimiter(rate.Every(1*time.Second), this.ratelimit)
	return this.iplimiter[ip]
}


// clearLimiter 清除IP限流器
// clearLimiter clear IP rate limiter
func (this *Apper) clearLimiter() {
	// 计算需要删除的数量（50%）
	// Calculate the number to delete (50%)
	removeCount := len(this.iplimiter) / 2
	for i:=0;i<removeCount;i++ {
		for k,_ := range this.iplimiter {
			delete(this.iplimiter, k)
		}
	}
	this.msg <- "清理限流列表50%的IP"
	// this.msg <- "Clear 50% of IPs from rate limit list"
}

// ServeHTTP 实现http.Handler接口，处理HTTP请求
// ServeHTTP implements the http.Handler interface to handle HTTP requests
func (this *Apper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	
	urlSpit := strings.Split(r.URL.String(), "?")
	ipaddr := this.GetClientIP(r)
	
	// 檢測url
	// if this.Bh.RegexUrl(r.URL.String()) {
	// 	this.Bh.Lock(ipaddr)
	// }
	// IP行爲判斷
	if this.Bh.CheckNotFound(ipaddr) {
		this.Bh.Lock(ipaddr)
	}
	if this.Bh.CheckScan(ipaddr) {
		this.Bh.Lock(ipaddr)
	}
	// IP是否鎖定
	if this.Bh.IsLock(ipaddr) {
		this.msg <- ipaddr+"被鎖定"
		http.Error(w, ipaddr+"被鎖定", http.StatusTooManyRequests)
		return
	}
	
	
	// * 限流處理
	// * Rate limiting processing
	if this.iplimiter!=nil{
		limiter:=this.getLimiter(ipaddr)
		if !limiter.Allow() {
			this.msg <- "限流"
			// this.msg <- "Rate limiting"
			http.Error(w, "限流", http.StatusTooManyRequests)
			// http.Error(w, "Rate limiting", http.StatusTooManyRequests)
			return
		}
	}

	// * 全局中間件處理
	// * Global middleware processing
	for _, g := range this.gMiddleware {
		err := g(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// * 路由中間件處理
	// * Route middleware processing
	if rs, ok := this.rMiddleware[r.Method][urlSpit[0]]; ok {
		for _, rone := range rs {
			err := rone(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	if h, ok := this.rmap[r.Method][urlSpit[0]]; ok {
		this.msg <- ipaddr + " " + r.Method + " " + r.URL.String() + " 200 OK"
		this.Bh.Record(ipaddr,200)
		h(w, r)
	} else {
		this.msg <- ipaddr + " " + r.Method + " " + r.URL.String() + " Not Found 404"
		this.Bh.Record(ipaddr,404)
		http.NotFound(w, r)
	}
}

// Run 启动HTTP服务器
// Run starts the HTTP server
func (this *Apper) Run() {
	go this.Logger()
	this.msg <- "apper HTTP is running in port:" + this.port
	// this.msg <- "apper HTTP is running in port:" + this.port

	server := &http.Server{Addr: ":" + this.port, Handler: this}
	server.ListenAndServe()
}

// RunTLS 启动HTTPS服务器
// RunTLS starts the HTTPS server
func (this *Apper) RunTLS(certFile, keyFile string) {
	go this.Logger()
	this.msg <- "apper HTTPS is running in port:" + this.port
	// this.msg <- "apper HTTPS is running in port:" + this.port

	server := &http.Server{Addr: ":" + this.port, Handler: this}
	server.ListenAndServeTLS(certFile, keyFile)
}