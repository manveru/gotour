package main

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/skratchdot/open-golang/open"
)

const host = "127.0.0.1:7879"

func main() {
	if len(os.Args) == 3 {
		pack(os.Args[1], os.Args[2])
		return
	}

	server()
	// go openBrowser()
	// gui()

}

func server() {
	println("Serving at " + host)

	tree := unpack("pano")
	// spew.Dump(tree)

	asset := func(path string) ([]byte, error) {
		if path == "" {
			path = "tour.html"
		}
		path = filepath.Clean("/" + path)
		if found, ok := tree[path]; ok {
			return found, nil
		}
		return nil, errors.New("not found")
	}

	assetDir := func(path string) ([]string, error) {
		fmt.Println("dir:", path)
		return nil, nil
	}

	http.Handle("/", http.FileServer(&AssetFS{Asset: asset, AssetDir: assetDir, Prefix: "/"}))
	http.ListenAndServe(host, nil)
}

func openBrowser() {
	url := "http://" + host + "/tour.html"
	tries := 0

	for {
		if tries > 100 {
			fmt.Println("giving up after ", time.Duration(tries)*100*time.Microsecond)
			os.Exit(1)
		}

		<-time.After(10 * time.Microsecond)

		if _, err := http.Get(url); err == nil {
			println("Opening " + url)
			fmt.Println(open.Start(url))
			return
		}

		tries++
	}
}

type MyMainWindow struct {
	*walk.MainWindow
}

func gui() {
	defer os.Exit(0)

	println("start gui")

	win := MainWindow{
		Title:  "Seitenschmied Virtual Tour",
		Layout: VBox{},
		Children: []Widget{
			PushButton{
				Text: "Server beenden",
				OnClicked: func() {
					os.Exit(0)
				},
			},
		},
	}

	win.Run()
}

func unpack(src string) map[string][]byte {
	tree := map[string][]byte{}

	fd, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	stat, err := fd.Stat()
	if err != nil {
		panic(err)
	}
	reader, err := zip.NewReader(fd, stat.Size())
	if err != nil {
		panic(err)
	}

	for _, f := range reader.File {
		rc, err := f.Open()
		if err != nil {
			panic(err)
		}
		all, err := ioutil.ReadAll(rc)
		if err != nil {
			panic(err)
		}

		tree[f.Name] = decrypt(all)
	}

	return tree
}

func pack(root string, dest string) {
	buf := bytes.Buffer{}
	zw := zip.NewWriter(&buf)

	root = filepath.Clean(root)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		_, name := filepath.Split(path)

		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			switch name {
			case ".git":
				return filepath.SkipDir
			case "go":
				return filepath.SkipDir
			}
		} else {
			if name == "tour.ptv" {
				return nil
			}

			f, err := zw.Create(strings.TrimPrefix(filepath.Clean(path), root))
			if err != nil {
				panic(err)
			}
			data, err := ioutil.ReadFile(path)
			if err != nil {
				panic(err)
			}
			_, err = f.Write(encrypt(data))
			if err != nil {
				panic(err)
			}
			zw.Flush()
		}

		return nil
	})

	if err := zw.Close(); err != nil {
		panic(err)
	}

	ioutil.WriteFile(dest, buf.Bytes(), 0644)
}

var commonIV = []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}
var key_text = "ganz sicherer key der unbedingt "

func encrypt(plaintext []byte) []byte {
	c, err := aes.NewCipher([]byte(key_text))
	if err != nil {
		fmt.Printf("Error: NewCipher(%d bytes) = %s", len(key_text), err)
		panic(err)
	}

	cfb := cipher.NewCFBEncrypter(c, commonIV)
	ciphertext := make([]byte, len(plaintext))
	cfb.XORKeyStream(ciphertext, plaintext)

	return ciphertext
}

func decrypt(ciphertext []byte) []byte {
	c, err := aes.NewCipher([]byte(key_text))
	if err != nil {
		fmt.Printf("Error: NewCipher(%d bytes) = %s", len(key_text), err)
		panic(err)
	}

	cfb := cipher.NewCFBDecrypter(c, commonIV)
	plaintext := make([]byte, len(ciphertext))
	cfb.XORKeyStream(plaintext, ciphertext)

	return plaintext
}

var fileTimestamp = time.Now()

// FakeFile implements os.FileInfo interface for a given path and size
type FakeFile struct {
	// Path is the path of this file
	Path string
	// Dir marks of the path is a directory
	Dir bool
	// Len is the length of the fake file, zero if it is a directory
	Len int64
}

func (f *FakeFile) Name() string {
	_, name := filepath.Split(f.Path)
	return name
}

func (f *FakeFile) Mode() os.FileMode {
	mode := os.FileMode(0644)
	if f.Dir {
		return mode | os.ModeDir
	}
	return mode
}

func (f *FakeFile) ModTime() time.Time {
	return fileTimestamp
}

func (f *FakeFile) Size() int64 {
	return f.Len
}

func (f *FakeFile) IsDir() bool {
	return f.Mode().IsDir()
}

func (f *FakeFile) Sys() interface{} {
	return nil
}

// AssetFile implements http.File interface for a no-directory file with content
type AssetFile struct {
	*bytes.Reader
	io.Closer
	FakeFile
}

func NewAssetFile(name string, content []byte) *AssetFile {
	return &AssetFile{
		bytes.NewReader(content),
		ioutil.NopCloser(nil),
		FakeFile{name, false, int64(len(content))}}
}

func (f *AssetFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("not a directory")
}

func (f *AssetFile) Size() int64 {
	return f.FakeFile.Size()
}

func (f *AssetFile) Stat() (os.FileInfo, error) {
	return f, nil
}

// AssetDirectory implements http.File interface for a directory
type AssetDirectory struct {
	AssetFile
	ChildrenRead int
	Children     []os.FileInfo
}

func NewAssetDirectory(name string, children []string, fs *AssetFS) *AssetDirectory {
	fileinfos := make([]os.FileInfo, 0, len(children))
	for _, child := range children {
		_, err := fs.AssetDir(filepath.Join(name, child))
		fileinfos = append(fileinfos, &FakeFile{child, err == nil, 0})
	}
	return &AssetDirectory{
		AssetFile{
			bytes.NewReader(nil),
			ioutil.NopCloser(nil),
			FakeFile{name, true, 0},
		},
		0,
		fileinfos}
}

func (f *AssetDirectory) Readdir(count int) ([]os.FileInfo, error) {
	if count <= 0 {
		return f.Children, nil
	}
	if f.ChildrenRead+count > len(f.Children) {
		count = len(f.Children) - f.ChildrenRead
	}
	rv := f.Children[f.ChildrenRead : f.ChildrenRead+count]
	f.ChildrenRead += count
	return rv, nil
}

func (f *AssetDirectory) Stat() (os.FileInfo, error) {
	return f, nil
}

// AssetFS implements http.FileSystem, allowing
// embedded files to be served from net/http package.
type AssetFS struct {
	// Asset should return content of file in path if exists
	Asset func(path string) ([]byte, error)
	// AssetDir should return list of files in the path
	AssetDir func(path string) ([]string, error)
	// Prefix would be prepended to http requests
	Prefix string
}

func (fs *AssetFS) Open(name string) (http.File, error) {
	name = path.Join(fs.Prefix, name)
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	if b, err := fs.Asset(name); err == nil {
		return NewAssetFile(name, b), nil
	}
	if children, err := fs.AssetDir(name); err == nil {
		return NewAssetDirectory(name, children, fs), nil
	} else {
		return nil, err
	}
}
