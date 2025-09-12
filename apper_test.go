package goweber

import (
	"fmt"
	"net/http"
	"testing"
	"errors"
)

func gMiddleware(r *http.Request) error {
	fmt.Println("啓用全局中間件")
	return nil
}

func rMiddleware1(r *http.Request) error {
	fmt.Println("啓用路由中間件1")
	return nil
}

func rMiddleware2(r *http.Request) error {
	fmt.Println("啓用路由中間件2")
	return errors.New("{'code': 500, 'message': '中間件2認證失敗'}")
}

// 處理緩存結構
type Chandler struct {
	Cache *Cacher
}
func (c *Chandler) Handler(w http.ResponseWriter, r *http.Request) {
	if c.Cache!=nil{
		if c.Cache.IsCache(w,r){
			fmt.Println("緩存中")
			return
		}
	}
	fmt.Fprintf(w, "緩存響應")
	if c.Cache!=nil{
		c.Cache.SetCache(r,1,"緩存響應，使用緩存")
	}
}

func TestApp(t *testing.T) {
	app := New()
	defer app.Close()

	// 基礎功能
	app.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})

	// 中間件測試
	app.Use(gMiddleware)
	
	app.Get("/middleware", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("中間件測試成功"))
	}, rMiddleware1, rMiddleware2)
	
	// 緩存處理
	ctest:=&Chandler{Cache:app.Cache}
	app.Get("/cache", ctest.Handler)


	fmt.Println(app)
	app.Run()
}
