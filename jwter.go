package goweber

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	// "fmt"
)

type Jwter struct {
	Rint    int
	Rstr    string
	Exphour int
	Version string
	Key     string
}

type Token struct {
	Id   string
	Exp  string
	Code string
}

func NewJwter() *Jwter {
	return &Jwter{Rint: 1, Rstr: "WHSS", Exphour: 8, Version: "V1", Key: "system"}
}

func (this *Jwter) Validate(token string) error {
	token, err := this.Decode(token)
	if err != nil {
		return err
	}

	// tokenSplit:=strings.Split(token,"-")
	tokenSplit, err := this.GetArr(token)
	if err != nil {
		return err
	}
	if len(tokenSplit) != 3 {
		return errors.New("token格式错误，解码失败")
	}

	code := this.Rstr + tokenSplit[0] + tokenSplit[1]
	hash := md5.Sum([]byte(code))
	codeString := hex.EncodeToString(hash[:])
	if codeString != tokenSplit[2] {
		return errors.New("token校验失败")
	}

	timestamp := int(time.Now().Unix())
	tokenstamp, err := strconv.Atoi(tokenSplit[1])
	if err != nil {
		return errors.New("token的过期时间校验失败")
	}
	if timestamp > tokenstamp {
		return errors.New("token过期")
	}
	this.Key = tokenSplit[0]
	return nil
}

func (this *Jwter) Decode(token string) (string, error) {
	if len(token) < 8+len(strconv.Itoa(this.Rint)) {
		return "", errors.New("token格式错误,长度不足")
	}
	tureToken := token[0 : len(token)-8-len(strconv.Itoa(this.Rint))]
	var encodeToken string
	for i := 0; i < len(tureToken)-1; i += 2 {
		num, err := strconv.ParseInt(string(tureToken[i])+string(tureToken[i+1]), 16, 64)
		if err != nil {
			return "", errors.New("token转码失败")
		}
		num = num - int64(this.Rint)
		encodeToken += string(rune(num))
		// encodeToken += strconv.Itoa(int(num))
	}
	return encodeToken, nil
}

func (this *Jwter) Encode() (string, error) {
	now := time.Now()
	h := now.Hour()
	i := now.Minute()
	d := now.Day()
	s := now.Second()
	hids := this.OneToTwo(strconv.Itoa(h)) + this.OneToTwo(strconv.Itoa(i)) + this.OneToTwo(strconv.Itoa(d)) + this.OneToTwo(strconv.Itoa(s))

	exp := strconv.Itoa(this.Exphour*60*60 + int(time.Now().Unix()))

	if strings.ContainsRune(this.Key, '-') {
		return "", errors.New("加密Key中不能出现-符号")
	}

	code := this.Rstr + this.Key + exp

	hash := md5.Sum([]byte(code))
	codeString := hex.EncodeToString(hash[:])

	jwt, err := this.GetJoin(this.Key, exp, codeString)
	if err != nil {
		return "", err
	}
	var token string
	for i := 0; i < len(jwt); i++ {
		// fmt.Println(int(jwt[i]))
		num := strconv.FormatInt(int64(jwt[i])+int64(this.Rint), 16)
		token += string(num)
	}

	rintstr := strconv.Itoa(this.Rint)

	// fmt.Println(token+string(this.Rint)+hids)
	// fmt.Println(token);
	// fmt.Println(this.Rint);
	// fmt.Println(hids);
	return token + rintstr + hids, nil
}

func (this *Jwter) GetJoin(key string, exp string, code string) (string, error) {
	if this.Version == "V1" {
		return key + "-" + exp + "-" + code, nil
	}
	if this.Version == "V2" {
		return "{\"Id\":\"" + key + "\",\"Exp\":\"" + exp + "\":\"Code\":\"" + code + "\"}", nil
	}
	return "", errors.New("jwt版本定义" + this.Version + "，当前不支持")
}

func (this *Jwter) GetArr(tokenstr string) ([]string, error) {
	var res []string
	if this.Version == "V1" {
		res = strings.Split(tokenstr, "-")
		return res, nil
	}
	if this.Version == "V2" {
		var t Token
		err := json.Unmarshal([]byte(tokenstr), &t)
		if err != nil {
			return res, errors.New("jwt V2解析token失败")
		}
		res = make([]string, 3)
		res[0] = t.Id
		res[1] = t.Exp
		res[2] = t.Code
		return res, nil
	}
	return res, errors.New("jwt版本定义" + this.Version + "，当前不支持")
}

func (this *Jwter) OneToTwo(val string) string {
	if len(val) == 1 {
		return "0" + val
	}
	return val
}
