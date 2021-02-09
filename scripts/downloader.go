// Copyright 2021 The golang.design Initiative Authors.
// All rights reserved. Use of this source code is governed
// by a GNU GPLv3 license that can be found in the LICENSE file.

// Quick solution, use it at your own risk
// written by Changkun Ou <changkun.de>
//
// This scripts tries to extract all image urls from github, download
// and save them to the ./assets folder, and then replaces all links
// from original github user content url to ../assets.
package main

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	re := regexp.MustCompile("https://user-images(.*)\\)")
	os.Mkdir("assets", os.ModePerm) // dont care error

	index := 0
	filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if !strings.Contains(info.Name(), ".md") {
			return nil
		}

		for {
			f, err := os.Open(info.Name())
			if err != nil {
				panic(err)
			}
			b, err := io.ReadAll(f)
			if err != nil {
				panic(err)
			}
			f.Close()
			content := string(b)
			links := re.FindAllStringIndex(string(b), -1)
			if len(links) == 0 {
				break
			}

			posStart := links[0][0]
			posEnd := links[0][1] - 1
			src := content[posStart:posEnd]
			dst := fmt.Sprintf("assets/%d.png", index)

			// 1. download image
			resp, err := http.Get(src)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			// 2. save in assets
			imgf, err := os.Create(dst)
			if err != nil {
				panic(err)
			}
			_, err = io.Copy(imgf, resp.Body)
			if err != nil {
				panic(err)
			}
			imgf.Close()

			// 3. replace links in doc
			content = content[:posStart] + "../" + dst + content[posEnd:]
			err = os.WriteFile(info.Name(), []byte(content), os.ModePerm)
			if err != nil {
				panic(err)
			}

			// 4. save the doc
			fmt.Println(src, dst)
			index++
		}
		return nil
	})
}
