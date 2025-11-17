/**
	安全策略，監控用戶的行爲。
	危險行爲定義：
	1.高頻請求
		- 限流處理，已實現，見apper.iplimiter
		- 掃描目錄，多次訪問不存在目錄，例如/project/user,/project/admin,404錯誤一個周期内超過50次，則封禁該ip
		- 遍歷爆破，例如：登錄接口多次頻繁嘗試密碼，采用時間間隔監控
	2.輸入内容異常
		- 異常參數：sql注入，XSS，命令注入，// ! 待完善
		- 提交大文件/數據，// ! 待完善
		
	方案：
	IP監控周期5分鐘，禁用規則
	1.404錯誤，超過50次
	2.10次内的訪問時間間隔相同，或少於1秒的次數超過7次
	3.url正則匹配，防止sql漏洞,XSS漏洞,命令 // ! 待完善
*/
package goweber

import (
	"time"
	"sync"
	"regexp"
	"math"
	// "fmt"
)

type Behaver struct { 
	IpMontion map[string]*IpInfo
	// 監控IP最大值
	IpMax int
	// 禁用IP
	IpDisable map[string]int64
	// 監控時間間隔
	Expire int64
	// 自動清理時間，分鐘
	CleanSecond int64
	mu sync.Mutex
}
type IpInfo struct { 
	Time int64
	NotFound int
	Times []int64
}
func NewBehaver() *Behaver { 
	bh:=&Behaver{
		IpMontion:make(map[string]*IpInfo),
		IpMax:1000,
		IpDisable:make(map[string]int64),
		Expire:300,
		CleanSecond:300,
	}
	// go bh.Clear()
	return bh
}

// * 防止Behaver數量溢出，每5分鐘清理
func (this *Behaver) Clear() { 
	timer := time.NewTicker(time.Duration(this.CleanSecond) * time.Second)
	for range timer.C { 
		this.mu.Lock()
		for k,v := range this.IpMontion { 
			if v.Time <= time.Now().Unix() - this.Expire {
				delete(this.IpMontion,k)
			}
		}
		for k,v := range this.IpDisable { 
			if v <= time.Now().Unix() {
				delete(this.IpDisable,k)
			}
		}
		this.mu.Unlock()
	}
}

// * 記錄該IP的行為
func (this *Behaver) Record(ip string,code int) { 
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.IpMax==0 {
		return
	}
	if _,ok:=this.IpMontion[ip]; ok { 
		this.IpMontion[ip].Times=append(this.IpMontion[ip].Times,time.Now().Unix())
	} else { 
		this.IpMontion[ip] = &IpInfo{
			Times:[]int64{time.Now().Unix()},
			NotFound:0,
			Time:time.Now().Unix(),
		}
	}
	if code == 404 {
		this.IpMontion[ip].NotFound++
	}
}

// * 鎖定IP
func (this *Behaver) Lock(ip string) { 
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.IpMax==0 {
		return
	}
	if _,ok:=this.IpMontion[ip];ok {
		delete(this.IpMontion,ip)
	}
	this.IpDisable[ip]=time.Now().Unix()
}

// * IP是否鎖定
func (this *Behaver) IsLock(ip string) bool { 
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.IpMax==0 {
		return false
	}
	if _,ok:=this.IpDisable[ip];ok {
		return true
	}
	return false
}

// * 檢查404錯誤
func (this *Behaver) CheckNotFound(ip string) bool { 
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.IpMax==0 {
		return false
	}
	if info,ok:=this.IpMontion[ip];ok { 
		if info.Time+this.Expire>time.Now().Unix() { 
			if info.NotFound>50 {
				return true
			}
		}
	}
	return false
}

// * 10次内訪問時間間隔相同
func (this *Behaver) CheckScan(ip string) bool { 
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.IpMax==0 {
		return false
	}
	if info,ok:=this.IpMontion[ip];ok { 
		if len(info.Times)>=10 {
			var time_split []int64
			same_split:=0
			// 獲得最近10次記錄
			for i := len(info.Times) - 10; i < len(info.Times)-1; i++ { 
				time_split=append(time_split,info.Times[i+1]-info.Times[i])
			}
			for i :=0;i<8;i++ {
				// time_split[i]-time_split[i+1]絕對值小於1
				diff := time_split[i+1] - time_split[i]
				if math.Abs(float64(diff)) <=1 { 
					same_split++
				}
			}
			// fmt.Println("time_split:",time_split)
			// fmt.Println("same_split:",same_split)
			// same_split>=7次，判斷
			if same_split>=8 {
				return true
			}
			
		}
	}
	return false
}

// * 正則匹配url
// ! 有問題，待完善
func (this *Behaver) RegexUrl(url string) bool { 
	if this.IpMax==0 {
		return false
	}
	// 正則匹配，防止sql漏洞,XSS漏洞,命令執行漏洞
	return regexp.MustCompile(`(?i)(?:union|sleep|benchmark|load_file|outfile|sleep|and|or|select|insert|delete|update|drop|create|alter|grant|exec|xp_cmdshell|net localgroup administrators|net user|net group administrators|net localgroup|net group|net user|cmd|whoami|systeminfo|tasklist|taskkill|reg query|reg add|reg delete|reg query|reg set|reg import|reg export|reg query|script)`).MatchString(url)
}