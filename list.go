package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"net/http"
	"io/ioutil"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"github.com/PuerkitoBio/goquery"
	"github.com/fatih/color"
)

var commonSkips = []string{"vendor", ".git", "examples", "node_modules"}
var projectMap = make(map[string]bool)
var currentProjectName string

type lister struct {
	root  string
	skips []string
	deps  map[string]string
	moot  *sync.Mutex
}

func List(pwd string, skips ...string) (map[string]string, error) {
	//pwd, _ := os.Getwd()
	l := lister{
		root:  pwd,
		skips: append(skips, commonSkips...),
		deps:  map[string]string{},
		moot:  &sync.Mutex{},
	}
	wg := &errgroup.Group{}

	components := strings.Split(pwd, "/")
	if len(components) > 1 {
		currentProjectName = components[len(components)-2] + "/" + components[len(components)-1]
	}
	color.Blue("---------------project-------------")
	GetDetail("github.com/" + currentProjectName)
	color.Blue("---------------imports-------------")
	err := filepath.Walk(pwd, func(path string, info os.FileInfo, err error) error {
		wg.Go(func() error {
			return l.process(path, info)
		})
		return nil
	})

	if err != nil {
		return l.deps, errors.WithStack(err)
	}

	err = wg.Wait()

	return l.deps, err
}

func (l *lister) add(dep string) {
	l.moot.Lock()
	defer l.moot.Unlock()
	l.deps[dep] = dep
}

func (l *lister) process(path string, info os.FileInfo) error {
	path = strings.TrimPrefix(path, l.root)
	if info.IsDir() {

		for _, s := range l.skips {
			if strings.Contains(strings.ToLower(path), s) {
				return nil
			}
		}

		cmd := exec.Command("go", "list", "-e", "-f", `'* {{ join .Deps  "\n"}}'`, "./"+path)
		b, err := cmd.Output()
		if err != nil {
			fmt.Println(string(b))
			return errors.WithStack(err)
		}

		list := strings.Split(string(b), "\n")

		for _, g := range list {
			//fmt.Println("-----", g)
			if strings.Contains(g, "'") {
				g = strings.TrimLeft(g, "'* ")
				g = strings.TrimRight(g, "'")
			}
			if strings.Contains(g, "vendor") {
				vendors := strings.Split(g, "vendor/")
				if len(vendors) > 1 {
					g = vendors[len(vendors)-1]
				}
			}
			if *verbose {
				if (strings.Contains(g, "github.com") || strings.Contains(g, "golang.org") || strings.Contains(g, "google.golang.org")) && !strings.Contains(g, currentProjectName) {
					g = strings.TrimPrefix(g, "'* ")
					l.add(g)
				}
			} else {
				if strings.Contains(g, "github.com") && !strings.Contains(g, currentProjectName) {
					g = strings.TrimPrefix(g, "'* ")
					l.add(g)
				}
			}

		}
	}
	return nil
}

func GetDoc(html string) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		fmt.Println(err)
	}
	return doc
}

func Title(doc *goquery.Document) string {
	title, _ := doc.Find("meta[property=\"og:description\"]").Attr("content")
	if title == "" {
		title = doc.Find("title").Text()
	}
	return title
}

func Stars(doc *goquery.Document) string {
	nodes := doc.Find(".social-count.js-social-count").Nodes
	if len(nodes) > 0 {
		nodedoc := goquery.NewDocumentFromNode(nodes[0])
		return strings.TrimSpace(nodedoc.Text())
	}
	return ""
}

func GetDetail(url string) {
	color.Green(url)
	components := strings.Split(url, "/")
	var projectUrl string
	if len(components) > 2 {
		projectUrl = "https://" + components[0] + "/" + components[1] + "/" + components[2]
		if ok := projectMap[projectUrl]; ok {
			return
		} else {
			projectMap[projectUrl] = true
		}
	}
	if projectUrl == "" || strings.Contains(projectUrl, "golang.org") || strings.Contains(projectUrl, "google.golang.org") {
		return
	}
	resp, err := http.Get(projectUrl)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	doc := GetDoc(string(body))
	title := Title(doc)
	if strings.Contains(title, "development by creating an account on GitHub") {
		color.Red("no description!")
	} else {
		fmt.Println(title)
	}
	color.Yellow(fmt.Sprintf("stars: %s", Stars(doc)))
}
