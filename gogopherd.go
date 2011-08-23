// This is a gopher server, written in Go.
//
// Author: Brad Fitzpatrick <bradfitz@golang.org>
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var root = flag.String("root", "", "root directory to serve")
var docdir string

func main() {
	flag.Parse()
	docdir = getDocDir()

	ln, err := net.Listen("tcp", ":70")
	if err != nil {
		log.Fatalf("Listen: %v", err)
	}
	for {
		c, err := ln.Accept()
		if err == os.EOF {
			return
		}
		go serve(c)
	}
}

func getDocDir() string {
	if *root != "" {
		fi, err := os.Stat(*root)
		if err != nil {
			log.Fatalf("failed to stat root %q: %v", *root, err)
		}
		if !fi.IsDirectory() {
			log.Fatalf("root %q isn't a directory", *root)
		}
		return *root
	}
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		log.Fatalf("No GOROOT enviroment variable defined.")
	}
	return filepath.Join(goroot, "doc")
}

func serve(c net.Conn) {
	defer c.Close()
	br := bufio.NewReader(c)
	bw := bufio.NewWriter(c)
	defer bw.Flush()

	sl, err := br.ReadSlice('\n')
	if err != nil {
		log.Printf("client readslice: %v", err)
		return
	}
	line := string(sl)
	if strings.HasSuffix(line, "\r\n") {
		line = line[:len(line)-2]
	}

	line = filepath.Clean(line)
	fileName := filepath.Join(docdir, line)
	fi, err := os.Stat(fileName)
	if err != nil {
		log.Printf("failed to stat %q: %v", fileName, err)
		return
	}
	switch {
	case fi.IsDirectory():
		f, err := os.Open(fileName)
		if err == nil {
			fis, err := f.Readdir(-1)
			sort.Sort(byFileName(fis))
			if err == nil {
				for _, fi := range fis {
					fmt.Fprintf(bw, "%s%s\r\n",
						itemType(&fi),
						strings.Join([]string{
							fi.Name,
							line + "/" + fi.Name,
							"127.0.0.1",
							"70",
						}, "\t"))
				}
			}
		}
	case fi.IsRegular():
		f, err := os.Open(fileName)
		if err != nil {
			log.Printf("Open: ", err)
			return
		}
		io.Copy(bw, f)
	default:
		log.Printf("unsupported file type with file %q", fileName)
	}
}

func itemType(fi *os.FileInfo) string {
	if fi.IsDirectory() {
		return "1"
	}
	name := fi.Name
	switch {
	case strings.HasPrefix(name, ".html"):
		return "h"
	case strings.HasPrefix(name, ".txt"):
		return "0"
	case strings.HasPrefix(name, ".gif"):
		return "g"
	case strings.HasPrefix(name, ".png"),
		strings.HasPrefix(name, ".jpg"),
		strings.HasPrefix(name, ".jpeg"):
		// TODO(bradfitz): re-encode pngs to gifs :)
		// For now, though:
		return "I" // Image file of unspecified format. Client decides
		// how to display. Often used for JPEG images
	}
	return "9" // binary file
}

type byFileName []os.FileInfo

func (s byFileName) Len() int { return len(s) }
func (s byFileName) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}
func (s byFileName) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
