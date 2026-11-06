package bridge

import (
	"embed"
	"io/fs"
)

//go:embed client/nodra.js client/nodra.d.ts
var clientFiles embed.FS

// Client holds the JavaScript client, nodra.js, and its types, nodra.d.ts, for apps that serve their pages from Go rather than build them with a bundler: serve it beside the pages, for example under /nodra/ with mygo.FileServer, and import "/nodra/nodra.js" from a module script. Apps with a bundler copy the two files into their frontend instead.
var Client fs.FS = mustSub(clientFiles, "client")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
