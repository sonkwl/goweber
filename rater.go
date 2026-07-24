// 限流器
// 基於IP的限流器，如果1秒内出現100請求都是404，則封禁IP 5分鐘
package goweber

import (
	"time"
)
// 限流器數據結構
type Rater struct {
	Start int // 是否開啓，0為關閉，1為開啓
    Second int // 監控秒
	BlockMinute int // 封禁分鐘
    ErrMax int // 最大請求錯誤次數
	IpMax int // 最大監控IP數量
	ErrorIps map[string]*IpData // 監控IP
	BlockIps map[string]time.Time // 封禁IP
}
// 監控IP數據結構
type IpData struct {
	Count int
	LastTime time.Time
}

// 實例化
func NewRater() *Rater {
	// 初始化，1秒出現10個404則封禁IP 5分鐘
	return &Rater{
		Start:0,
		Second: 1,
		ErrMax: 10,
		IpMax: 10000,
		BlockMinute: 5,
		ErrorIps: make(map[string]*IpData),
		BlockIps: make(map[string]time.Time),
	}
}

// 判斷IP監控狀態
func (this *Rater) SetStatus(ip string) {
	if this.Start==0{
        return
	}
	// 超過監控上綫，清理
    if len(this.ErrorIps)>this.IpMax{
        this.ClearErrorIps()
    }
    if ipData,exists:=this.ErrorIps[ip];exists{
        if time.Since(ipData.LastTime)>time.Duration(this.Second)*time.Second{
            ipData.Count=1
			ipData.LastTime=time.Now()
        }else{
            ipData.Count++
			if ipData.Count>=this.ErrMax{
			    this.BlockIps[ip]=time.Now() // 加入鎖定列表
				delete(this.ErrorIps,ip) //從監控列表刪除
			}
		}
    }else{
        this.ErrorIps[ip]=&IpData{Count:1,LastTime:time.Now()} // 加入監控列表
	}
}

// 判斷是否鎖定中
func (this *Rater) IsBlocked(ip string) bool {
	if this.Start==0 {
        return false
	}
    // * 測試使用
    // fmt.Println(this.ErrorIps)
    // fmt.Println(this.BlockIps)
	// 超過監控上綫，清理
    if len(this.ErrorIps)>this.IpMax{
        this.ClearBlockIps()
    }
    if _,exists:=this.BlockIps[ip];exists{
        if time.Since(this.BlockIps[ip])>=time.Duration(this.BlockMinute) * time.Minute {
            delete(this.BlockIps,ip) // 解鎖
            return false
        }
        return true
    }
    return false
}

// * 監控列表清理
func (this *Rater) ClearErrorIps() {
    for ip,ipData:=range this.ErrorIps{
        if time.Since(ipData.LastTime)>=time.Duration(this.Second) * time.Second {
            delete(this.ErrorIps,ip)
        }
    }
}
// * 鎖定列表清理
func (this *Rater) ClearBlockIps() {
    for ip,_:=range this.BlockIps{
        if time.Since(this.BlockIps[ip])>=time.Duration(this.BlockMinute)*time.Minute{
            delete(this.BlockIps,ip)
        }
    }
}