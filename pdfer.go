package goweber

import (
	"bytes"
	"fmt"
	"os"
	"unicode/utf16"
	"image/jpeg"
	"errors"
)

/**
A4紙一頁PDF
*/

type Pdfer struct{
	Buf bytes.Buffer
	Offsets []int // offsets of each page
	Offset int // current offset
	X int // current x position
	Y int // current y position
	StreamData string // stream data for current page
	ImgData []string // image data for current page
	ImgMask []int // image mask for current page
	Font string //font style
	Err error // error
}

func NewPdfer() *Pdfer {
	return &Pdfer{
		Offsets: []int{},
		Offset: 0,
		X: 0,
		Y: 0,
		StreamData: "",
		ImgData: []string{},
		ImgMask: []int{1,0,1,0,1,0},
		Font: "STSong-Light",
	}
}
// * 寫入obj
func (p *Pdfer) WriteObj(objID int, content string) {
    p.Offsets = append(p.Offsets, p.Buf.Len())
	fmt.Fprintf(&p.Buf, "%d 0 obj\n%s\nendobj\n", objID, content)
}
// * 構建文字數據
func (p *Pdfer) WriteText(str string,x int,y int,fs int,fw int,x_algin int) {
	// 自動設置段落位置
	if y>0{
		p.Y = y
	}
	// 轉換成u16格式
	u16:= utf16.Encode([]rune(str))
	// 每行字數
	max:=int(500/fs)
	// 獲得文字占用x軸空間
	str_x:= len(u16)*fs
	// x_algin:0 默認居左，1居中，2，居右
	if x_algin==1 {
		x=x+int((500-str_x)/2)
	} else if x_algin==2 {
		x=x+int(500-str_x)
	}
	hexArr:=[]string{} // str行拆分
	asc_white:=0 // 英文數字字間距控制
	charArray:="" // 拆分後的str
	i:=0
	for _,r:=range u16 {
		// 英文字母或數字，直接轉換
		if r<128 {
			// cid的寬度默認1000
			charArray += fmt.Sprintf("<%04X> 500 ", r)
			// 2個英文算1個漢字寬度
			asc_white++
			if asc_white==2 {
				asc_white=0
				i++
			}
		}else{
			charArray += fmt.Sprintf("<%04X> 0 ", r)
			i++
		}
		// 一行滿
		if i==max {
		    hexArr=append(hexArr,charArray)
		    charArray=""
		    i=0
		}
	}
	hexArr=append(hexArr,charArray) // 最後一行
	// 遍歷hexArr,賦值streamData
	for _,hexStr:=range hexArr {
		p.StreamData+=fmt.Sprintf(
			"BT\n"+
			"%d %d Td\n"+
			"/ChineseFont %d Tf\n"+
			"%d Tr\n"+
			"[%s] TJ\n"+
			"ET\n",
		x,p.Y,fs,fw,hexStr)
		p.Y=p.Y-fs-10 //行間距
	}
	p.Y-=10 // 段落間距
}

// * 構建圖片數據
func (p *Pdfer) WriteImage(imgPath string,x int,y int,w int,h int) {
	// 讀取文件數據
    imgData,err:=os.ReadFile(imgPath)
	if err!=nil {
		p.Err=errors.New("讀取圖片失敗:"+err.Error())
		return
	}
	// 使用 Go 原生 image/jpeg 獲取圖片的像素寬高
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(imgData))
	if err != nil {
		p.Err=errors.New("解析圖片失敗:"+err.Error())
		return
	}
	// 構建streamdata 圖片部分
	imgcount:=len(p.ImgData) // 第幾個圖片
	p.StreamData+=fmt.Sprintf(
		"q\n"+
		"%d 0 0 %d %d %d cm\n"+
		"/Img%d\n"+
		"Do\nQ\n",
	w,h,x,y,imgcount+1)
	
	// ImgMax是否定義正確
	if len(p.ImgMask)!=6 {
		p.Err=errors.New("ImgMask定義錯誤")
		return
	}
	// 每個圖片構建ImgData
	p.ImgData=append(p.ImgData,fmt.Sprintf(
		"<<\n"+
		"/Type /XObject\n"+
		"/Subtype /Image\n"+
		"/Width %d\n"+
		"/Height %d\n"+
		"/ColorSpace /DeviceRGB\n"+
		"/BitsPerComponent 8\n"+
		"/Filter /DCTDecode\n"+
		"/Length %d\n"+
		"/Mask [%d %d %d %d %d %d]\n"+
		">>\n"+
		"stream\n%s\nendstream\n",
	cfg.Width,cfg.Height,len(imgData),p.ImgMask[0],p.ImgMask[1],p.ImgMask[2],p.ImgMask[3],p.ImgMask[4],p.ImgMask[5],imgData))
	// 每個圖片構建ImgDict
	// p.ImgDict=append(p.ImgDict,fmt.Sprintf(
	// 	"/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode",
	// cfg.Width,cfg.Height))
}

// * 寫入PDF obj 1,2部分
func (p *Pdfer) WriteHeader() {
    p.Buf.WriteString("%PDF-1.4\n")
	p.WriteObj(1,fmt.Sprintf("<<\n/Type /Catalog\n/Pages 2 0 R\n>>"))
	p.WriteObj(2,fmt.Sprintf("<<\n/Type /Pages\n/Kids [3 0 R]\n/Count 1\n>>"))
}

// * 寫入PDF obj 3:page,4:stream
func (p *Pdfer) WritePage() {
    // 先拼接imgdata部分
	imgdata:=""
	for k,_:=range p.ImgData {
		// 4文字内容,5文字font,6文字格式,7文字cmap,8開始存圖片相關配置
		imgdata+=fmt.Sprintf("/Img%d %d 0 R\n",k+1,k+8) 
	}
	if imgdata!="" {
		imgdata="/XObject <<\n"+imgdata+"\n>>\n"
	}
	p.WriteObj(3,
		"<<\n"+
			"/Type /Page\n"+
			"/Parent 2 0 R\n"+
			"/MediaBox [0 0 595.28 841.89]\n"+
			"/Contents 4 0 R\n"+
			"/Resources <<\n"+
				imgdata+
				"/Font <<\n/ChineseFont 5 0 R\n>>\n"+
			">>\n"+
		">>\n",
	)
	p.WriteObj(4,fmt.Sprintf(
		"<< /Length %d >>\n"+
		"stream\n%s\nendstream\n",
	len(p.StreamData),p.StreamData))
}

//* 寫入中文映射表，放在obj 5,6,7
func (p *Pdfer) WriteCmap() {
	p.WriteObj(5,fmt.Sprintf(
		"<<\n"+
			"/Type /Font\n"+
			"/Subtype /Type0\n"+
			"/BaseFont /%s\n"+
			"/Encoding /UniGB-UTF16-H\n"+
			"/DescendantFonts [6 0 R]\n"+
			"/ToUnicode 7 0 R\n"+
		">>\n",
	p.Font))
	p.WriteObj(6,fmt.Sprintf(
		"<<\n"+
			"/Type /Font\n"+
			"/Subtype /CIDFontType0\n"+
			"/BaseFont /%s\n"+
			"/CIDSystemInfo <<\n"+
				"/Registry (Adobe)\n"+
				"/Ordering (GB1)\n"+
				"/Supplement 5\n"+
			">>\n"+
		">>\n",
	p.Font))
	cmap:=`/CIDInit /ProcSet findresource begin
			12 dict begin
			begincmap
			/CIDSystemInfo <<
			/Registry (Adobe)
			/Ordering (UCS)
			/Supplement 0
			>> def
			/CMapName /Custom-ToUnicode def
			/CMapType 2 def
			1 begincodespacerange
			<0000> <FFFF>
			endcodespacerange
			1 beginbfrange
			<0000> <FFFF> <0000>
			endbfrange
			endcmap
			CMapName currentdict /CMap defineresource pop
			end
			end
			`
	p.WriteObj(7,fmt.Sprintf(
		"<< /Length %d >>\n"+
		"stream\n%s\nendstream\n",
	len(cmap),cmap))
}
//* 寫入圖片obj 從8開始
func (p *Pdfer) WriteImageData() {
    for k,obj:=range p.ImgData {
        p.WriteObj(k+8,obj)
    }
}
// * 寫入pdf結尾部分
func (p *Pdfer) WriteFooter() {
    xrefOffset := p.Buf.Len()
	totalObjects := len(p.Offsets) + 1
	fmt.Fprintf(&p.Buf, "xref\n0 %d\n", totalObjects)
	fmt.Fprintf(&p.Buf, "0000000000 65535 f \n")
	for _, offset := range p.Offsets {
		fmt.Fprintf(&p.Buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&p.Buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", totalObjects, xrefOffset)
}

// * 輸出PDF
func (p *Pdfer) Save(filename string) {
	p.WriteHeader()
	p.WritePage()
	p.WriteCmap()
	p.WriteImageData()
	p.WriteFooter()
    err:=os.WriteFile(filename,p.Buf.Bytes(),0644)
	if err!=nil {
		p.Err=errors.New("寫入PDF失敗:"+err.Error())
		return
	}
}