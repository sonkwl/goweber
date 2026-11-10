package goweber

import (
	"fmt"
	// "net/http"
	"testing"
	// "errors"
)

func TestPdfer(t *testing.T) {
	pdf:=NewPdfer()
	pdf.WriteText("个人信息保护告知书",50,750,24,2,1) //x,y,fontsize,fontweight,x_algin,y=0自動
	
	pdf.WriteText("尊敬的应聘者：",50,0,14,0,0)
	pdf.WriteText("    感谢您选择应聘本公司，根据《中华人民共和国个人信息保护法》等相关法律法规的规定，为保障您的个人信息权益，本公司基于人力资源管理之招聘作业所必需，下面请您填写或录入的应聘表单中个人信息相关内容，此信息仅用于本次招聘作业简历筛选、面试评估及录用流程，请您确保提供之信息的真实性、准确性。本公司承诺将采取合理的安全管理措施，保护您的个人信息安全，防止信息泄露、篡改、丢失、滥用或被未经授权访问。未经您同意或法律许可，公司不会向第三方提供您的个人信息。",50,0,14,0,0)
	pdf.WriteText("    若您未被录用，本公司按照《公司个人信息保护管理规定》将您的个人信息资料删除或销毁；",50,0,14,0,0)
	pdf.WriteText("    若您被录用，本公司会将您的个人信息资料存档保存。",50,0,14,0,0)
	pdf.WriteText("√本人已阅读并理解本《个人信息保护告知书》的全部内容，同意贵公司在招聘过程中依法收集、使用和处理我的个人信息。",50,0,14,0,0)
	pdf.WriteText("應聘者：           ",50,200,14,0,2)
	pdf.WriteText("日期：2026年10月14日",50,0,14,0,2)
	
	pdf.WriteImage("./pdfstatic/logo.jpg",50,800,105,23) //x,y,width,height
	pdf.ImgMask=[]int{250,255,250,255,250,255}
	pdf.WriteImage("./pdfstatic/name.jpg",400,200,120,70)
	pdf.WriteImage("./pdfstatic/seal.jpg",320,80,200,200)
	
	pdf.Save("./pdfstatic/test.pdf")
	if pdf.Err!=nil {
		fmt.Println(pdf.Err)
		return
	}
	fmt.Println("pdf生成成功")
}