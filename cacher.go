package goweber

import (
	"fmt"
	"net/http"
	"sync"
	"time"
	"unsafe"
)

type Cacher struct {
	cacheData   map[string]string //緩存數據
	cacheTime   map[string]int64  //緩存時間
	maxSize     int64             //緩存大小
	currentSize int64             //當前緩存大小
	mu          sync.RWMutex      // 添加读写锁
}

// * 新建緩存,m為空間大小，單位 m
func NewCacher(m int64) *Cacher {
	return &Cacher{cacheData: make(map[string]string), cacheTime: make(map[string]int64), maxSize: m * 1024 * 1024, currentSize: 0}
}

// * 根據url定義緩存時間和緩存數據,m為分鐘
func (this *Cacher) SetCache(r *http.Request, m int64, data string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.cacheData == nil {
		return
	}
	if this.cacheTime == nil {
		return
	}
	// * 緩存大小超過最大值，刪除過期緩存
	if this.currentSize > this.maxSize {
		// * 當前時間
		currentTime := time.Now().Unix()
		// * 嘗試刪除過期緩存
		expired := false
		// * 遍歷緩存
		for k, exp := range this.cacheTime {
			if exp < currentTime {
				delete(this.cacheTime, k)
				delete(this.cacheData, k)
				this.currentSize -= int64(unsafe.Sizeof(data))
				// expired = true
				continue
			}
		}
		// * 過期緩存不存在，清空緩存
		if expired == false {
			this.cacheTime = make(map[string]int64)
			this.cacheData = make(map[string]string)
			this.currentSize = 0
		}
	}
	// * 設置緩存
	url := r.URL.String()
	this.cacheTime[url] = time.Now().Unix() + m*60
	this.cacheData[url] = data
	this.currentSize += int64(unsafe.Sizeof(data))
}

// * 判斷緩存是否存在，存在則返回緩存數據
func (this *Cacher) IsCache(w http.ResponseWriter, r *http.Request) bool {
	this.mu.Lock()
	defer this.mu.Unlock()
	// * 只處理get請求
	if r.Method != "GET" {
		return false
	}
	// * 獲得url
	url := r.URL.String()
	
	if this.cacheData == nil {
		return false
	}
	if this.cacheTime == nil {
		return false
	}

	if exp, ok := this.cacheTime[url]; ok {
		if exp > time.Now().Unix() {
			if data, ok := this.cacheData[url]; ok {
				fmt.Fprint(w, data)
				return true
			}
			return false
		} else {
			// * 緩存過期刪除
			delete(this.cacheTime, url)
			delete(this.cacheData, url)
		}
	}
	return false
}

// * 獲取占用空間大小
func (this *Cacher) Size() int64 {
	return this.currentSize
}
