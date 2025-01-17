package helpers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/alasdairforsythe/norm"
	"github.com/go-resty/resty/v2"
	"github.com/gookit/goutil"
	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/slog"
	"github.com/iancoleman/strcase"
	"github.com/olekukonko/tablewriter"
	"github.com/samber/lo"
	"github.com/tdewolff/minify/v2"
	mhtml "github.com/tdewolff/minify/v2/html"
	"github.com/yosssi/gohtml"
	gh "gitlab.dev.ict/golang/libs/gohttp"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/gonet"
	"gitlab.dev.ict/golang/libs/utils"
	"golang.org/x/net/html"
)

const (
	MT_HTML = "text/html"
)

var (
	rnnn        = regexp.MustCompile(`\n{3,}`)
	symblsEnd   = regexp.MustCompile(`\s+$`)
	symblsStart = regexp.MustCompile(`^\s+`)
	rn          = regexp.MustCompile(`\n+`)
	rt          = regexp.MustCompile(`\t+`)
	rntsStart   = regexp.MustCompile(`^[ \t\n]+`)
	rntsEnd     = regexp.MustCompile(`[ \t\n]+$`)
	rs          = regexp.MustCompile(" +")
	rEmptyLines = regexp.MustCompile(`(?m)^\s*$`) // pattern for a line containing only '\n,\t,\n,space' ((?m) - multiline mode)
)

var ff = fmt.Printf
var minifyHtml *minify.M

func init() {
	minifyHtml = minify.New()
	minifyHtml.Add(MT_HTML, &mhtml.Minifier{
		KeepDefaultAttrVals: true,
		KeepDocumentTags:    true,
		KeepComments:        false,
		KeepWhitespace:      false,
	})
}

type Result struct {
	URL          string
	Title        string
	TextContent  string
	OriginalPage string
	Err          error
}

// String of result
func (r *Result) String() string {
	return fs("Result{url=%s, title=[%s], len(textContent)=%d, len(OriginalPage)=%d, error=%v}", r.URL, r.Title, len(r.TextContent), len(r.OriginalPage), r.Err)
}

type ProcessSelectionFN func(s *goquery.Selection) *goquery.Selection
type SkipElementFN func(*goquery.Selection) bool

// SkipElemWithClass returns a function SkipElement to skip elements that have one of the given classes.
func SkipElemWithClass(class ...string) SkipElementFN {
	return func(s *goquery.Selection) bool {
		for _, v := range class {
			if s.HasClass(v) {
				return true
			}
		}
		return false
	}
}

// SkipElemWithNodeNames returns a function SkipElement to skip an element if it matches one of the given node names.
func SkipElemWithNodeNames(htmlElements ...string) SkipElementFN {
	return func(s *goquery.Selection) bool {
		for _, v := range htmlElements {
			if goquery.NodeName(s) == v {
				return true
			}
		}
		return false
	}
}

// ExcludeScripts(implements ProcessSelectionFN) removes script, noscript, head, and style elements from the given goquery.Selection.
// It returns the modified goquery.Selection.
func ExcludeScripts(s *goquery.Selection) *goquery.Selection {
	s.Find("script").Remove()
	s.Find("noscript").Remove()
	s.Find("head").Remove()
	s.Find("style").Remove()
	return s
}

// FindBodyAndExcludeScript implements ProcessSelectionFN to return the body of the document and exclude script tags from them.
func FindBodyAndExcludeScript(s *goquery.Selection) *goquery.Selection {
	return s.Find("body").Children().Not("script").Not("noscript")
}

func ScrapAndParse(log *slog.Record, rc *resty.Client, url string, isDebug bool, arrProcessSelectionFN []ProcessSelectionFN, skipArr ...SkipElementFN) (res *Result) {
	log.Info("Start for url:", url)
	var respBytes []byte
	var err error
	if strings.HasPrefix(url, "file") {
		log.Infof("cwd=[%s] ReadFile=[%s]", lo.Must(os.Getwd()), strings.TrimPrefix(url, "file://"))
		respBytes, err = os.ReadFile(strings.TrimPrefix(url, "file://"))
	} else {
		resp, e := rc.R().SetDebug(isDebug).SetContext(log.Ctx).Get(url)
		if e != nil {
			err = fmt.Errorf("http error: %v", e)
		}
		respBytes = resp.Body()
	}

	if err != nil {
		log.Errorf("Error while get url [%s] >> %v", url, err)
		return &Result{URL: url, Err: err}
	}

	// respBytes, e = m.Bytes("text/html", resp.Body())
	// if e != nil {
	// respBytes = resp.Body()
	// panic(e)
	// }
	d, _ := goquery.NewDocumentFromReader(bytes.NewReader(gohtml.FormatBytes(respBytes)))
	if d.Find("body").Children().Length() <= 1 {
		log.Warnf("Stop, cause body is empty: %s", url)
		return &Result{URL: url, Err: fmt.Errorf("body is empty or contains only one element or captcha")}
	}

	var selection = d.Selection
	for _, fn := range arrProcessSelectionFN {
		selection = fn(selection)
	}
	if isDebug {
		log.Debugf("BODY_BEFORE_CLEANING:\n%s", respBytes)
		log.Debug(goquery.OuterHtml(selection))
	}
	return &Result{URL: url,
		Title:        cleanExtraTabCharsOnEdges(log, d.Selection.Find("title").Text()),
		TextContent:  HTMLParse(log, selection, isDebug, skipArr...),
		OriginalPage: string(respBytes)}
}

func ParseWebPageByURLResty(ctx context.Context, log *gl.Logger, url string, isDebug bool, arrProcessSelectionFN []ProcessSelectionFN, skipArr ...SkipElementFN) (res *Result) {
	rec := log.WithCtx(ctx)
	restyCl := gonet.NewRestyClient(log)
	resp, e := restyCl.R().SetContext(ctx).Get(url)
	if e != nil {
		return nil
	}
	d, _ := goquery.NewDocumentFromReader(bytes.NewReader(gohtml.FormatBytes(resp.Body())))

	var selection = d.Selection
	for _, fn := range arrProcessSelectionFN {
		selection = fn(selection)
	}
	return &Result{URL: url,
		Title:        cleanExtraTabCharsOnEdges(rec, d.Selection.Find("title").Text()),
		TextContent:  ParseWebPage(ctx, selection, isDebug, skipArr...),
		OriginalPage: string(resp.Body())}
}

// The HTMLParse function processes and transforms an HTML document into a markdown-like string format.
// It performs various operations based on the parameters received, such as logging, skipping elements, and cleaning extra characters.
// The function accepts a *slog.Record for logging purposes, a *goquery.Selection to parse the HTML,
// a boolean `isDebug` to enable or disable debug logging, and a variadic slice of SkipElementFN functions to skip certain elements.
// The function returns a string, which represents the processed HTML in a simplified markdown-like format.
func HTMLParse(log *slog.Record, s *goquery.Selection, isDebug bool, skipArr ...SkipElementFN) string {
	log.Infof("Start parse html. isDebug=%t skipArr.len=%d", isDebug, len(skipArr))
	t := time.Now()

	var buf bytes.Buffer   // a buffer used to build the result string
	var f func(*html.Node) // a recursive function to process each node in the HTML document

	f = func(n *html.Node) {
		gqNode := goquery.NewDocumentFromNode(n)

		// skip elements based on the provided skip functions
		for _, fn := range skipArr {
			if fn(gqNode.Selection) {
				return
			}
		}

		// in debug mode, log detailed information about each node
		if isDebug {
			log.Debugf(">>> HTML_NODE:[type=%d name=%s]; Parent:[%v] DATA=[%s] ATTR=[%s] OutherHTML=[%s]",
				n.Type, goquery.NodeName(gqNode.Selection), n.Parent, n.Data, n.Attr, utils.StrCut(goutil.Must(goquery.OuterHtml(gqNode.Selection)), 200))
		}

		// process text nodes that are not children of 'a' or 'script' elements
		if n.Type == html.TextNode && n.Parent.Data != "a" && n.Parent.Data != "script" {
			if str := strings.TrimSpace(n.Data); str != "" {
				log.Debugf("[if n.Type == html.TextNode] => WriteString: %s; CLEANED:[%s]", str, cleanExtraTabChars(str))
				buf.WriteString(cleanExtraTabChars(str))
			}
		}

		// process different node types based on their tag name
		txt := gqNode.Text()
		switch n.Data {
		case "a":
			txt = symblsEnd.ReplaceAllString(cleanExtraTabChars(txt), "")
			log.Debugf("[CASE [a]] => WriteString: [%s]; CLEANED:[%s]", strconv.Quote(gqNode.Text()), strconv.Quote(txt))
			if href, ok := gqNode.Attr("href"); ok && !strings.HasPrefix(href, "javascript") {
				buf.WriteString(fs(" %s(%s)", txt, href))
			} else {
				buf.WriteString(fs(" %s", txt))
			}
			return
		case "p", "br":
			buf.WriteString("\n")
		case "table":
			log.Debug("Meet <TABLE>")
			buf.WriteString("\n")
			buf.WriteString(processTable(log, gqNode.Selection))
			buf.WriteString("\n")
			return
		case "tr":
			buf.WriteString("\n")
		case "td", "th":
			buf.WriteString(" | ")
		case "li":
			buf.WriteString("\n- ")
		case "h1", "h2", "h3", "h4", "h5", "h6":
			buf.WriteString(fs("\n\n# %s\n", cleanExtraTabCharsFull(log, txt)))
			return
		case "code":
			txt := gqNode.Text()

			if !strings.Contains(txt, "\n") && !strings.Contains(txt, " ") {
				buf.WriteString(fs(" `%s`", txt))
			} else {
				buf.WriteString(fs("\n```%s```\n", txt))
			}
			return
		case "summary":
			buf.WriteString(fs("\n> %s:", gqNode.Text()))
			return
		}

		if n.FirstChild != nil {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
	}
	for _, n := range s.Nodes {
		f(n)
	}
	log.WithData(gl.M{"elapsed": time.Since(t).Milliseconds()}).Infof("End parse html")

	return cleanExtraTabCharsOnEdges(log, buf.String())
}

// ParseWebPage parses a web page using the provided goquery.Selection and returns the extracted text.
// It takes a context.Context, a *goquery.Selection, a boolean flag isDebug, and an optional variadic parameter skipArr of type SkipElementFN.
// Iterates over the HTML nodes and extracts the text based on the node type and parent data.
// If isDebug is true, debug logs are printed.
// The extracted text is appended to a buffer and returned after cleaning extra tab characters.
func ParseWebPage(ctx context.Context, s *goquery.Selection, isDebug bool, skipArr ...SkipElementFN) string {
	return HTMLParse(gl.New(gl.WithLevel(gl.LevelInfo), gl.WithOC()).WithCtx(ctx), s, isDebug, skipArr...)
}

func RemoveExcessEmptyLines(input string) string {
	re := regexp.MustCompile(`(?m)^\s*\r?\n{3,}`)
	return re.ReplaceAllString(input, "\n\n")
}

func TrimLines(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	foundFirstNonEmptyLine := false
	var sb strings.Builder
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" && !foundFirstNonEmptyLine {
			continue
		}
		foundFirstNonEmptyLine = true
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func processTable(log *slog.Record, s *goquery.Selection) string {
	formatText := func(input string) string {
		lines := strings.Split(input, "\n")
		var result string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				if result != "" && result[len(result)-1] != ':' {
					result += " "
				}
				result += trimmed
			}
		}
		return result
	}

	// rgn := regexp.MustCompile(`\n{1,}`)
	var buf bytes.Buffer
	// csv := csv.NewWriter(&buf)
	// csv.Comma = '|'
	var data [][]string
	s.Find("tbody").Each(func(_ int, table *goquery.Selection) {
		rows, cells := table.Find("tr").Length(), table.Find("tr").First().Find("th, td").Length()
		log.Tracef("--- Total lines: %d cells=%d\n", rows, cells)

		// check if exists colaspan attr and then add to cells
		table.Find("tr").First().Find("th, td").Each(func(i int, s *goquery.Selection) {
			if colspan, ok := s.Attr("colspan"); ok {
				colspanCountCells, _ := strconv.Atoi(colspan)
				cells += colspanCountCells - 1
			}
		})
		log.Infof("--- Total lines: %d cells=%d\n", rows, cells)

		for i := 0; i < rows; i++ {
			data = append(data, make([]string, cells))
		}
		// csv.WriteAll(data)
		log.Tracef("data = %v\n", buf)
		table.Find("tr").Each(func(rowIndex int, row *goquery.Selection) {
			row.Find("th, td").Each(func(j int, cell *goquery.Selection) {
				// if panic, write html element to log
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("Table parsing error on [%d:%d]: PANIC: %v", rowIndex, j, r)
						log.Errorf(">> HTML CURRENT \nCELL:\n %s\nROW:\n %s;\nTABLE:\n %s", selectionToMinifiedStr(cell), selectionToMinifiedStr(row), selectionToMinifiedStr(table))
					}
				}()

				rowSpan, spanExists := cell.Attr("rowspan")
				text := strings.TrimSpace(cell.Text())
				log.Tracef(">>>[%d:%d] name=%s rowspan=%s text=[%s]", rowIndex, j, goquery.NodeName(cell), rowSpan, text)

				if rn.MatchString(text) {
					text = formatText(text)
				}

				if spanExists {
					rowSpanCountRows, _ := strconv.Atoi(rowSpan)
					for i := rowIndex; i < rowIndex+rowSpanCountRows; i++ {
						log.Tracef("****** [%d:%d] also add to [%d:%d]", rowIndex, j, i, j)
						data[i][j] = text
					}
				} else {
					if data[rowIndex][j] != "" {
						j = j + 1
					}
					data[rowIndex][j] = text
				}
			})
			log.Tracef("> rowData = %v\n", data[rowIndex])
		})
	})

	if len(data) == 1 {
		return data[0][0]
	}

	tw := tablewriter.NewWriter(&buf)
	tw.SetHeader(data[0])
	formatTbl2(tw)

	tw.AppendBulk(data[1:])
	tw.Render()
	return buf.String()
}

func selectionToMinifiedStr(s *goquery.Selection) string {
	if h, e := goquery.OuterHtml(s); e != nil {
		return ""
	} else {
		return lo.Must(minifyHtml.String(MT_HTML, h))
	}
}

func formatTbl1(tw *tablewriter.Table) {
	tw.SetAutoWrapText(true)
	tw.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	tw.SetNoWhiteSpace(true)
	// tw.SetReflowDuringAutoWrap(true)
	tw.SetAutoFormatHeaders(false)
	// tw.SetColumnSeparator("|")
	tw.SetCenterSeparator("|")
	tw.SetHeaderLine(true)
}
func formatTbl2(tw *tablewriter.Table) {
	tw.SetAutoWrapText(false)
	tw.SetAutoFormatHeaders(true)
	tw.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tw.SetAlignment(tablewriter.ALIGN_LEFT)
	tw.SetNoWhiteSpace(false)
	tw.SetAutoFormatHeaders(false)
	tw.SetHeaderLine(true)
	tw.SetBorder(false)
}
func formatTbl(tw *tablewriter.Table) {
	tw.SetAutoWrapText(false)
	tw.SetAutoFormatHeaders(true)
	tw.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tw.SetAlignment(tablewriter.ALIGN_LEFT)
	tw.SetCenterSeparator("")
	tw.SetColumnSeparator("")
	tw.SetRowSeparator("")
	tw.SetHeaderLine(false)
	tw.SetBorder(false)
	tw.SetTablePadding("\t") // pad with tabs
	tw.SetNoWhiteSpace(true)
}

func cleanExtraTabChars(s string) (res string) {
	res = rnnn.ReplaceAllString(s, "\n\n")
	res = rt.ReplaceAllString(res, "\t")
	res = rs.ReplaceAllString(res, " ")
	return
}

func cleanExtraTabCharsOnEdges(rec *slog.Record, s string) (res string) {
	log := func(name, s string, l int) {
		rec.Debugf("%s: - [%s...%s]", name, utils.StrCut(s, l), utils.StrCutEnd(s, l))
	}
	log("TEXT", s, 10)
	res = rntsStart.ReplaceAllString(s, "")
	log("TEXT_rntsStart", res, 10)
	res = rntsEnd.ReplaceAllString(res, "")
	log("TEXT_rntsEnd", res, 10)
	return
}

// CleanEmptyLine - remove empty lines. e.g.: \n \n \n -> \n\n
func CleanEmptyLine(s string) (res string) {
	res = rEmptyLines.ReplaceAllString(s, "")
	return
}

func cleanExtraTabCharsFull(rec *slog.Record, s string) (res string) {
	res = cleanExtraTabChars(s)
	res = cleanExtraTabCharsOnEdges(rec, res)
	return
}

func isValidURL(u string) bool {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return false
	}
	return parsedURL.Scheme == "http" || parsedURL.Scheme == "https"
}

func GetAllLinksFromSel(baseUrl string, s *goquery.Selection) map[string]string {
	if !isValidURL(baseUrl) {
		return nil
	}
	linksMap := GetAllLinksAsMap(s)
	inUrl, _ := url.Parse(baseUrl)

	for k, v := range linksMap {
		if !strings.HasPrefix(k, "http") && !strings.Contains(k, "~") {
			delete(linksMap, k)
			k = inUrl.ResolveReference(goutil.Must(url.Parse(k))).String()
		}
		if isValidURL(k) {
			linksMap[k] = ToSnake(v)
		}
	}
	return linksMap
}

// GetAllLinks - get all links from url
func GetAllLinks(u string, processSelectionFN ProcessSelectionFN) map[string]string {
	r, e := gh.Default().Client.Get(u)
	goutil.PanicErr(e)
	d, _ := goquery.NewDocumentFromReader(r.Body)
	return GetAllLinksFromSel(u, processSelectionFN(d.Selection))
}

// GetAllLinkAsMaps - get all links from html
func GetAllLinksAsMap(s *goquery.Selection) (m map[string]string) {
	m = make(map[string]string)
	s.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			m[href] = strings.TrimSpace(s.Text())
		}
	})
	return
}

func GetAllLinksAsArr(s *goquery.Selection, postProcessFN ...func(string) string) []string {
	var links []string
	s.Find("a").Each(func(i int, s *goquery.Selection) {
		if href, ok := s.Attr("href"); ok {
			links = append(links, href)
		}
	})
	return links
}

func GetAllLinksHTTPArr(mapLinks map[string]string) []string {
	var links []string
	for _, v := range maputil.Values(mapLinks) {
		l := strutil.MustString(v)
		if strings.HasPrefix(l, "http") {
			links = append(links, l)
		}
	}
	return links
}

// ToSnake converts a string to snake_case.
func ToSnake(s string) string {
	reg := regexp.MustCompile(`[\p{L}\d][\p{L}\d\s\.\(\)-]*[\p{L}\d\.()]`)
	s = strings.NewReplacer("(", "", ")", "").Replace(s)
	name := reg.Find([]byte(s))
	if len(name) == 0 {
		return ""
	}
	return strcase.ToSnake(string(name))
}

func NormalizeMonster(s string) string {
	nmlz, err := norm.NewNormalizer("nfd lines collapse trim")
	if err != nil {
		panic(err)
	}
	normStr, err := nmlz.Normalize([]byte(s))
	if err != nil {
		panic(err)
	}
	return string(normStr)
}

func GetDomainFromUrl(urlStr string) string {
	re := regexp.MustCompile(`(?i)//([^/.]+)/?`)
	return re.FindStringSubmatch(urlStr)[1]
}

func Normalize(in string) string {
	in = strings.Join(strutil.SplitTrimmed(in, "\n"), "\n")
	return in
}

// RemoveSysTags - remove script tags
func RemoveSysTags(s *goquery.Selection) *goquery.Selection {
	s.RemoveFiltered("script")
	return s
}

// RemoveDoubleLines - remove double lines
func RemoveDoubleLines(s string) string {
	regex := regexp.MustCompile(`\n{2,}`)
	return regex.ReplaceAllString(s, "\n")
}
