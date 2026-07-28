// * 暴力破解監控
// * 舉例：同一個對/login，在一段時間内報錯多次，判定暴力破解，禁用IP 24小時
// * 設定：1分鐘內，10次錯誤，則禁用IP 24小時
package goweber

import (
	"time"
	"sync"
	"net/http"
	"net"
)

type Bruter struct {
	// 1分鐘內，10次錯誤，則禁用IP 24小時
	MaxIp int // 監控IP數量
	MaxErr int // 監控錯誤數量
	MaxTime int // 監控時間秒
	ErrIps   map[string]*BruterIP
	BlockIps map[string]time.Time
	sync.RWMutex
}

type BruterIP struct {
	IP string
	Err int
	LastTime time.Time
}

func NewBruter() *Bruter {
    return &Bruter{
		MaxIp: 10000,
		MaxErr: 10,
		MaxTime: 60,
		ErrIps: make(map[string]*BruterIP),
		BlockIps:make(map[string]time.Time),
    }
}

// 獲得IP
func (this *Bruter) GetClientIP(r *http.Request) string {
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

// * 檢查IP是否被禁用
func (this *Bruter) IsBlocked(ip string) bool {
    this.RLock()
	defer this.RUnlock()
	if len(this.BlockIps) > this.MaxIp {
		this.Clear()
	}
	if _, ok := this.BlockIps[ip]; ok {
		// 是否鎖定超24小時
		if this.BlockIps[ip].After(time.Now()) {
			return true
		}
		delete(this.BlockIps, ip)
		return false
	}
	return false
}
// * 改變IP狀態
func (this *Bruter) SetStatus(ip string) {
    this.Lock()
	defer this.Unlock()
	if len(this.ErrIps) > this.MaxIp {
		this.Clear()
	}
	if _, ok := this.ErrIps[ip]; !ok {
		this.ErrIps[ip] = &BruterIP{
			IP: ip,
			Err: 1,
			LastTime: time.Now(),
		}
	} else {
		this.ErrIps[ip].Err++
	}
	if time.Since(this.ErrIps[ip].LastTime) > time.Duration(this.MaxTime)*time.Second {
		this.ErrIps[ip].Err = 1
		this.ErrIps[ip].LastTime = time.Now()
	}
	if this.ErrIps[ip].Err >= this.MaxErr {
		this.BlockIps[ip] = time.Now().Add(24 * time.Hour)
		delete(this.ErrIps, ip)
	}
}

// * 清除過期IP
func (this *Bruter) Clear() {
    this.Lock()
	defer this.Unlock()
	for ip, _ := range this.ErrIps {
		if time.Since(this.ErrIps[ip].LastTime) > time.Duration(this.MaxTime)*time.Second {
			delete(this.ErrIps, ip)
		}
	}
	for ip, _ := range this.BlockIps {
		if time.Since(this.BlockIps[ip]) > 24 * time.Hour {
			delete(this.BlockIps, ip)
		}
	}
}