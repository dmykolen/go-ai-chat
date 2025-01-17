package pdf

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/ledongthuc/pdf"
	"github.com/stretchr/testify/assert"

	"github.com/gofiber/fiber/v2/log"
)

const PDF1 = "../../_testdata/bzip2.pdf"
const PDF2 = "../../_testdata/2.pdf"

func TestPDFLoad(t *testing.T) {
	content, title, err := PDFLoad(PDF1)
	t.Logf("title: %s; err: %+v", title, err)
	// if assert.NoError(t, err) && assert.NotEmpty(t, content) {
	t.Logf("content: %s", strconv.Quote(content))
	// }
}

func TestReadPdf0(t *testing.T) {
	pdf.DebugOn = true
	f, p, err := pdf.Open(PDF1)
	assert.NoError(t, err)
	defer f.Close()
	// var (
	// 	reader io.Reader
	// 	b      []byte
	// )

	// reader, err = p.GetPlainText()
	// assert.NoError(t, err)

	// b, err := io.ReadAll(reader)
	// assert.NoError(t, err)

	log.Warn("PDF.Trailer.Keys: ", p.Trailer().Keys())
	log.Warn("PDF.Outline().Title: ", p.Outline().Title)
	log.Warn("p.Page(1).V.String():", p.Page(1).V.String())
	log.Info("#######################")
	log.Warn(p.NumPage())
	log.Info("#######################")
	log.Warn(p.Outline())
	log.Info("#######################")
	for i, ch := range p.Outline().Child {
		log.Warn(i, ch.Title, ch.Child)
		log.Info("#######################")

	}
	log.Warn()
	log.Info("#######################")
	log.Warn(p.Page(1).V.Name())
	log.Info("#######################")
	log.Warn(p.Trailer())
	log.Info("#######################")
	log.Warn(p.Trailer().Name())
	log.Info("#######################")
	log.Warn(p.Trailer().Key("Title"))
	log.Warn(p.Trailer().Key("Info"))
	log.Info("#######################")
	log.Warn(p.Trailer().Keys())
	log.Info("#######################")
	log.Warn(p.Trailer().Key("Info").Key("Author"))
	log.Info("#######################")
	log.Warn(p.Trailer().Key("ID"))
	log.Warn(p.Trailer().Key("ID").String())
	log.Warn(p.Trailer().Key("ID").RawString())
	log.Warn(p.Trailer().Key("ID").Text())
	log.Warn(p.Trailer().Key("ID").Kind())
	log.Warn(p.Trailer().Key("ID").TextFromUTF16())
	log.Warn(p.Trailer().Key("ID").Index(0).TextFromUTF16())
	log.Info("#######################")
	log.Info("#######################")
	log.Info("#######################")
	rows, err := p.Page(1).GetTextByRow()
	for _, cont := range rows[0].Content {
		fmt.Println(strconv.Quote(cont.S))
	}
	// log.Warn(rows[0].Content[0].S)
	log.Info("#######################")
	log.Info("#######################")
	log.Warn()
}
func TestReadPdf(t *testing.T) {
	s, err := readPdf3(PDF1)
	if assert.NoError(t, err) {
		t.Logf("content: %s", s)
	}
}

func readPdf2(path string) (string, error) {
	f, r, err := pdf.Open(path)
	// remember close file
	defer f.Close()
	if err != nil {
		return "", err
	}
	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}
		var lastTextStyle pdf.Text
		texts := p.Content().Text
		for _, text := range texts {
			lastTextStyle.S = text.S
			fmt.Printf("Font: %s, Font-size: %f, x: %f, y: %f, content: %s \n", lastTextStyle.Font, lastTextStyle.FontSize, lastTextStyle.X, lastTextStyle.Y, lastTextStyle.S)
			lastTextStyle = text
		}
	}
	return "", nil
}

func readPdf3(path string) (string, error) {
	f, r, err := pdf.Open(path)
	defer func() {
		_ = f.Close()
	}()
	if err != nil {
		return "", err
	}
	totalPage := r.NumPage()

	for pageIndex := 1; pageIndex <= totalPage; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		rows, _ := p.GetTextByRow()
		for _, row := range rows {
			println(">>>> row: ", row.Position)
			for _, word := range row.Content {
				fmt.Println(word.S)
			}
		}
	}
	return "", nil
}
