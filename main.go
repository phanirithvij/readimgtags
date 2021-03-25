package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
)

// Copied from the internal map in the file where tiff.DataType is defined
var typeNames = map[tiff.DataType]string{
	tiff.DTByte:      "byte",
	tiff.DTAscii:     "ascii",
	tiff.DTShort:     "short",
	tiff.DTLong:      "long",
	tiff.DTRational:  "rational",
	tiff.DTSByte:     "signed byte",
	tiff.DTUndefined: "undefined",
	tiff.DTSShort:    "signed short",
	tiff.DTSLong:     "signed long",
	tiff.DTSRational: "signed rational",
	tiff.DTFloat:     "float",
	tiff.DTDouble:    "double",
}

// https://stackoverflow.com/a/60500680/8608146
type printer struct {
	fname string
	done  bool
}

func (p *printer) Walk(name exif.FieldName, tag *tiff.Tag) error {
	if tag.Type == tiff.DTByte || tag.Type == tiff.DTSByte || tag.Type == tiff.DTAscii {
		// print filname for the very first time
		if !p.done {
			fmt.Println(p.fname)
			p.done = true
		}
		str := string(tag.Val)
		// https://stackoverflow.com/a/54285884/8608146
		if name == exif.UserComment {
			// https://www.awaresystems.be/imaging/tiff/tifftags/privateifd/exif/usercomment.html
			fmt.Println(name, typeNames[tag.Type], string(tag.Val[8:]), hex.EncodeToString(tag.Val[:8]))
			str = string(tag.Val[8:])
		}
		r := strings.NewReplacer("\x00", "")
		str = r.Replace(str)
		fmt.Printf("%40s: %s\n", name, str)
	}
	return nil
}

// returns if file is a valid image, any error
func printExif(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return true, err
	}
	if !isImage(path) {
		return false, nil
	}
	defer f.Close()
	x, err := exif.Decode(f)
	if err != nil {
		// not an image or doesnt have exif data
		// TODO logger warn or error
		if errors.Is(err, io.EOF) {
			return true, err
		}
		log.Println(err)
		return true, err
	}
	p := &printer{fname: path}
	x.Walk(p)
	return true, nil
}

func isImage(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		log.Println(err)
		return true
	}
	defer f.Close()
	buff := make([]byte, 512) // docs tell that it take only first 512 bytes into consideration
	if _, err = f.Read(buff); err != nil {
		fmt.Println(err) // do something with that error
		return false
	}
	return strings.HasPrefix(http.DetectContentType(buff), "image/")
}

func mainx() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fname := os.Args[1]
	f, err := os.Open(fname)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if i, err := f.Stat(); err == nil {
		if i.IsDir() {
			// TODO look at https://github.com/karrick/godirwalk
			filepath.Walk(fname, func(path string, info os.FileInfo, err error) error {
				if info.Mode().IsRegular() {
					printExif(path)
				}
				return nil
			})
		} else {
			printExif(fname)
		}
	}
}
