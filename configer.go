package goweber

import (
	"bufio"
	"os"
	"strings"
)

type Configer struct {
	params map[string]map[string]string
	file   *os.File
}

func (this *Configer) SetFile(f *os.File) {
	this.file = f
	this.ReadFile()
}

func (this *Configer) ReadFile() error {
	scanner := bufio.NewScanner(this.file)
	title := "gobal"
	for scanner.Scan() {
		line := []byte(scanner.Text())
		switch {
		case len(line) == 0:
			continue
		case line[0] == '#': //注释过滤
			continue
		case line[0] == ';': //注释过滤2
			continue
		case line[0] == '[':
			title = string(line)[1 : len(line)-1] //title赋值
			continue
		default:
			//分割key:value
			kv := strings.Split(string(line), "=")
			if len(kv) == 1 {
				continue
			}
			key := strings.Trim(kv[0], " ") //去除空格
			val := strings.Trim(kv[1], " ") //去除空格
			val = strings.Trim(val, "\r\n") //去除window换行符
			val = strings.Trim(val, "\n")   //去除unix换行符

			if this.params == nil {
				this.params = make(map[string]map[string]string)
			}
			if this.params[title] == nil {
				this.params[title] = make(map[string]string)
			}
			this.params[title][key] = val //赋值map
		}
	}
	if err := scanner.Err(); err != nil {
		this.file.Close()
		return err
	}
	this.file.Close()
	return nil
}

func (this *Configer) Get(title string, key string) string {
	if val, ok := this.params[title][key]; ok {
		return val
	}
	return ""
}
