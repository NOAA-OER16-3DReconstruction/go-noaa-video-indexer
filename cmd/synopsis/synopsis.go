package main

import (
	"flag"
	"github.com/amarburg/go-movieset"
	"log"
	// "strings"
	"fmt"
	"github.com/bamiaux/rez"
	"html/template"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

var count uint

func main() {

	var source, outdir string
	var step uint64
	scaleFactor := float32(0.2)

	flag.StringVar(&outdir, "outdir", "_html", "Directory for resulting html files")
	flag.Uint64Var(&step, "step", uint64(math.Trunc(29.97*60)), "")

	flag.Parse()

	if len(flag.Args()) == 0 {
		log.Fatalf("Must specify a movieset .json file on the command line")
	}
	source = flag.Args()[0]

	mm, err := movieset.LoadMultiMov(source)
	if err != nil {
		log.Fatalf("Unable to load movieset from \"%s\": %s", source, err)
	}

	outpath := filepath.Join(".", outdir)
	os.MkdirAll(outpath, os.ModePerm)
	imagedir := filepath.Join(outpath, "images")
	os.MkdirAll(imagedir, os.ModePerm)
	thumbdir := filepath.Join(outpath, "thumbs")
	os.MkdirAll(thumbdir, os.ModePerm)

	thumbnailFileName := func(idx uint64) string {
		return filepath.Join(thumbdir, fmt.Sprintf("thumb_%08d.png", idx))
	}

	imageFileName := func(idx uint64) string {
		return filepath.Join(imagedir, fmt.Sprintf("frame_%08d.png", idx))
	}

	makeImages := func(frameNum uint64) {
		_, errImg := os.Stat(imageFileName(frameNum))
		_, thumbImg := os.Stat(thumbnailFileName(frameNum))
		if errImg == nil && thumbImg == nil {
			log.Printf("Skipping %s", imageFileName(frameNum))
			return
		}

		img, _ := mm.ExtractFrame(frameNum)
		imgFile, _ := os.Create(imageFileName(frameNum))
		png.Encode(imgFile, img)
		imgFile.Close()

		thumb := image.NewRGBA(image.Rect(0, 0,
			int(float32(img.Bounds().Dx())*scaleFactor),
			int(float32(img.Bounds().Dy())*scaleFactor)))
		rez.Convert(thumb, img, rez.NewBicubicFilter())
		thumbFile, _ := os.Create(thumbnailFileName(frameNum))
		png.Encode(thumbFile, thumb)
		thumbFile.Close()
	}

	indexfile := filepath.Join(outpath, "index.html")
	outfile, err := os.Create(indexfile)
	if err != nil {
		log.Fatalf("Unable to open the output file \"%s\": %s", indexfile, outfile)
	}
	defer outfile.Close()

	const (
		forall   = `{{range .Sequence}}{{template "TableRow" .}}{{"\n"}}{{end}}`
		tablerow = `{{define "TableRow"}}<tr><td>{{template "ImageInfo" .}}</td><td>{{template "ImageTable" .}}</td></tr>{{end}}`
		imginfo  = `{{define "ImageInfo"}}{{movName .}}{{end}}`
		imgtable = `{{define "ImageTable"}}<table><tr>{{template "ImageElem" .}}</tr></table>{{end}}`
		imgelem  = `{{define "ImageElem"}}{{range framesIn .}}<td>{{template "ImageCell" .}}</td>{{end}}{{end}}`
		imgcell  = `{{define "ImageCell"}}<a href="{{ imageName .}}"><img src="{{ thumbnailName .}}"></a><br>{{.}}{{end}}`
	)

	// type imgcelldata struct {
	//   FrameNum uint64
	//   Hash     movieset.MovHash
	// }

	var funcs = template.FuncMap{"framesIn": func(seq movieset.SequenceElement) []uint64 {
		//log.Printf("%T %#v %d", hash, hash, count)
		start := uint64(math.Trunc(float64(seq.FrameOffset)/float64(step))) * step
		mov, _ := mm.Movies[seq.Hash]
		stop := uint64(math.Trunc(float64(seq.FrameOffset+mov.NumFrames)/float64(step))) * step

		//out := make([]imgcelldata, 0, int(math.Trunc(float64(stop-start)/float64(step))))
		out := make([]uint64, 0, int(math.Trunc(float64(stop-start)/float64(step))))

		for i := start; i < stop; i += step {
			//  out = append(out,imgcelldata{ FrameNum: i, Hash: seq.Hash })
			if i == 0 {
				out = append(out, 1)
			} else {
				out = append(out, i)
			}
		}
		return out
	},
		"movName": func(seq movieset.SequenceElement) string {
			return mm.Movies[seq.Hash].ShortName
		},
		"thumbnailName": func(frameNum uint64) string {
			path, _ := filepath.Rel(outpath, thumbnailFileName(frameNum))
			return path
		},
		"imageName": func(frameNum uint64) string {
		makeImages(frameNum)
		path, _ := filepath.Rel(outpath, imageFileName(frameNum))
		return path
		},
	}

	fmt.Fprintf(outfile, "<html><head><title>%s</title></head>\n", filepath.Base(source))
	fmt.Fprintf(outfile, "<table>\n")

	tableTempl := template.New("table").Funcs(funcs)
	if _, err = tableTempl.Parse(tablerow); err != nil {
		log.Fatal(err)
	}
	if _, err = tableTempl.Parse(forall); err != nil {
		log.Fatal(err)
	}
	if _, err = tableTempl.Parse(imginfo); err != nil {
		log.Fatal(err)
	}
	if _, err = tableTempl.Parse(imgtable); err != nil {
		log.Fatal(err)
	}
	if _, err = tableTempl.Parse(imgelem); err != nil {
		log.Fatal(err)
	}
	if _, err = tableTempl.Parse(imgcell); err != nil {
		log.Fatal(err)
	}
	if err != nil {
		log.Fatal(err)
	}

	if err := tableTempl.Execute(outfile, mm); err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(outfile, "</table>")
	fmt.Fprintf(outfile, "</body></html>")

}
