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
	cache:=app.Cache
	app.Get("/cache", func(w http.ResponseWriter, r *http.Request) {
		if cache != nil {
			if cache.IsCache(w,r) {
				fmt.Println("緩存中")
				return
			}
		}
		fmt.Fprintf(w, "緩存響應")
		if cache!=nil{
			cache.SetCache(r,1,"緩存響應，使用緩存")
		}
	})
	
	// 处理jwt
	jwter:=app.Jwt
	app.Get("/jwt/get", func(w http.ResponseWriter, r *http.Request) {
		jwter.Key="F6987445"
		token, err := jwter.Encode()
		if err != nil {
			w.Write([]byte(err.Error()))
		} else {
			w.Write([]byte(token))
		}
	})
	app.Get("/jwt/check",func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		err:=jwter.Validate(token)
		if err != nil {
			w.Write([]byte("jwt驗證失敗,err:"+err.Error()))
			return	
		}
		w.Write([]byte("jwt驗證成功,key:"+jwter.Key))
	 })

	// * 测试文件上传
	uploader := app.File
	app.Post("/upload", func(w http.ResponseWriter, r *http.Request) { 
		uploader.FieldName = "s_file"
		uploader.FieldNames = "s_files"
		uploader.Keyword = "F6987445"
		// * 处理文件上传
		savePaths,err:=uploader.HandleUpload(r)
		if err != nil {
			w.Write([]byte("文件上传失败,err:"+err.Error()))
			return
		}
		fmt.Fprintf(w, "文件上传成功，保存路径: %v", savePaths)
	})

	fmt.Println(app)
	app.Run()
}
