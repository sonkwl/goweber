package goweber

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type FileUploader struct {
	MaxSize      int64
	AllowedTypes []string
	SavePath     string
	FieldName    string // 添加单文件字段名
	FieldNames   string // 添加多文件字段名
	Keyword      string // 添加文件名关键字
}

func NewFileUploader(maxSize int64, allowedTypes []string, savePath string) *FileUploader {
	// 如果没有提供allowedTypes，则使用默认值
	if len(allowedTypes) == 0 {
		allowedTypes = []string{".pdf", ".xlsx", ".txt", ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".mp4", ".doc", ".docx", ".ppt", ".pptx"}
	}

	return &FileUploader{
		MaxSize:      maxSize,
		AllowedTypes: allowedTypes,
		SavePath:     savePath,
		FieldName:    "file",  // 默认单文件字段名
		FieldNames:   "files", // 默认多文件字段名
		Keyword:      "default",
	}
}

// HandleUpload 修改为支持批量文件上传
func (f *FileUploader) HandleUpload(r *http.Request) ([]string, error) {
	// 检查请求方法
	if r.Method != "POST" {
		return nil, fmt.Errorf("只允许POST方法")
	}

	// 检查内容长度
	if r.ContentLength > f.MaxSize {
		return nil, fmt.Errorf("文件过大，最大允许%d字节", f.MaxSize)
	}

	// 解析表单
	err := r.ParseMultipartForm(f.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("解析表单失败")
	}

	// 获取所有文件 (使用自定义字段名)
	files := r.MultipartForm.File[f.FieldNames]
	if len(files) == 0 {
		// 兼容单文件上传 (使用自定义字段名)
		file, header, err := r.FormFile(f.FieldName)
		if err != nil {
			return nil, fmt.Errorf("获取文件失败")
		}
		defer file.Close()

		// 处理单个文件
		savePath, err := f.saveSingleFile(file, header)
		if err != nil {
			return nil, err
		}
		return []string{savePath}, nil
	}

	// 处理多个文件
	var savedPaths []string
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return nil, fmt.Errorf("打开文件失败: %s", fileHeader.Filename)
		}

		savePath, err := f.saveSingleFile(file, fileHeader)
		file.Close() // 确保文件关闭

		if err != nil {
			return nil, fmt.Errorf("保存文件 %s 失败: %v", fileHeader.Filename, err)
		}
		savedPaths = append(savedPaths, savePath)
	}

	return savedPaths, nil
}

// 保存单个文件的辅助方法
func (f *FileUploader) saveSingleFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// 检查文件类型
	if !f.isAllowedType(header.Filename) {
		return "", fmt.Errorf("不支持的文件类型: %s", header.Filename)
	}

	// 生成新的文件名：时间戳+关键字+扩展名
	ext := filepath.Ext(header.Filename)
	timestamp := time.Now().Unix()

	// 创建新的文件名
	newFilename := fmt.Sprintf("%d_%s%s", timestamp, f.Keyword, ext)
	savePath := filepath.Join(f.SavePath, newFilename)

	// 创建文件
	dst, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %s", savePath)
	}
	defer dst.Close()

	// 复制文件
	_, err = io.Copy(dst, file)
	if err != nil {
		return "", fmt.Errorf("保存文件失败: %s", savePath)
	}

	return savePath, nil
}

func (f *FileUploader) isAllowedType(filename string) bool {
	if len(f.AllowedTypes) == 0 {
		return true
	}

	ext := filepath.Ext(filename)
	for _, allowedType := range f.AllowedTypes {
		if ext == allowedType {
			return true
		}
	}
	return false
}
