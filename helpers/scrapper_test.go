package helpers

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
	gl "gitlab.dev.ict/golang/libs/gologgers"
	"gitlab.dev.ict/golang/libs/gonet"
	lu "gitlab.dev.ict/golang/libs/utils"
)

const (
	htmlCode = `
		<body>
		<p>This is test script
		<code>
		$(document).ready(function(){
			$("button").click(function(){
				$("p").hide();
			});
		});
		</code>
		</p>
		<a href="https://example.com">Link 1</a>
		<div><p>Par 3</p></div>
		<a href="https://example.com">Link 2</a>
		<a href="https://example.org">Link 3</a>
		</body>`
	html1 = `
	<html>
		<body>
			<p>Paragraph 1</p>
			<ul>
				<li>Item 1</li>
				<li>Item 2</li>
			</ul>
			<div><p>Paragraph 2</p></div>
			<div class="empty"></div>
			<div class="onlytext">DimaDimo4ka</div>
			<div>DimaDimo4ka website - <a class="testLink" href="http://google.com">LINK</a></div>
			<p>DimaDimo4ka website - <a class="testLink" href="http://google.com">LINK</a></p>
			DimaDimo4ka in BODY website - <a class="testLink" href="http://google.com">LINK</a>
			<table>
				<tr>
					<td>Data 1</td>
					<td>Data 2</td>
				</tr>
				<tr>
					<td>Data 3</td>
					<td>Data 4</td>
				</tr>
			</table>
			<p>Paragraph 3</p>
		</body>
	</html>
	`
	html2 = `
	<html>
		<head></head>
		<body>
			    <!-- <link rel='stylesheet' type='text/css' media='screen' href='main.css'> -->
    <!-- <script src='main.js'></script> -->
			<div class="menu">
				<ul class="ul-list">
					<li class="li-item">Item 1</li>
					<li class="li-item">Item 2</li>
				</ul>
			</div>
			<div class="content">
				<div class="header">THIS IS HEADER</div>
				<div class="menu">THIS IS Menu</div>
				<div class="main">Boooooooooooody</div>
				<div class="footer">THIS IS FOOTER</div>
			</div>
			<script type="text/javascript">
				jQuery(function($){
    			// default state
    			$('.tariff-description-tab').addClass('selected');
    			$('.tariff-description').show();
				});
			</script>
		</body>
	</html>
	`
	html3 = `
<html>
<head>
	<title>Test</title>
	<style>
		.content {
			width: 100%;
			height: 100%;
			background-color: #f1f1f1;
		}

		.header {
			width: 100%;
			height: 50px;
			background-color: #f1f1f1;
		}

		.menu {
			width: 100%;
			height: 50px;
			background-color: #f1f1f1;
		}
	</style>
	<script type="text/javascript">
		jQuery(function ($) {
			// default state
			$('.tariff-description-tab').addClass('selected');
			$('.tariff-description').show();
		});
	</script>
</head>
<body>
	<!-- <link rel='stylesheet' type='text/css' media='screen' href='main.css'> -->
	<div class="menu">
		<ul class="ul-list">
			<li class="li-item">Item 1</li>
			<li class="li-item">Item 2</li>
		</ul>
	</div>
	<style>
		.content {
			width: 100%;
			height: 100%;
			background-color: #f1f1f1;
		}

		.header {
			width: 100%;
			height: 50px;
			background-color: #f1f1f1;
		}

		.menu {
			width: 100%;
			height: 50px;
			background-color: #f1f1f1;
		}
	</style>
	<!-- <script src='main.js'></script> -->
	<div class="content">
		<div class="header">THIS IS HEADER</div>
		<div class="menu">THIS IS Menu</div>
		<div class="main">
			<p>Boooooooooooody<p>
			<code>
			$(document).ready(function(){
				$("button").click(function(){
					$("p").hide();
				});
			});
			</code>
		</div>
		<div class="footer">THIS IS FOOTER</div>
	</div>
	<script type="text/javascript">
		jQuery(function ($) {
			// default state
			$('.tariff-description-tab').addClass('selected');
			$('.tariff-description').show();
		});
	</script>
</body>
</html>
`
)

func help_print(t *testing.T, s string) {
	sep := "##########################################################################################"
	t.Logf("%s\n%s\n%s", sep, s, sep)
}

func Test_ExcludeScripts(t *testing.T) {
	d, _ := goquery.NewDocumentFromReader(strings.NewReader(html3))
	before := lo.Must(goquery.OuterHtml(d.Selection))
	before_num_scripts := d.Find("script").Length()
	before_num_styles := d.Find("style").Length()
	before_num_head := d.Find("head").Length()

	ExcludeScripts(d.Selection)
	help_print(t, selectionToMinifiedStr(d.Selection))
	testhelp_printDiffAsTable(before, lo.Must(goquery.OuterHtml(d.Selection)), false)

	t.Logf("BEFORE: scripts=%d, styles=%d, head=%d", before_num_scripts, before_num_styles, before_num_head)
	t.Logf("AFTER : scripts=%d, styles=%d, head=%d", d.Find("script").Length(), d.Find("style").Length(), d.Find("head").Length())

	assert.Empty(t, d.Find("script").Nodes)
	assert.Empty(t, d.Find("style").Nodes)
	assert.Empty(t, d.Find("head").Nodes)
}

func TestHtml5(t *testing.T) {
	selector := "div.navigation-section"
	rc := gonet.NewRestyClient(gl.New(gl.WithOC(), gl.WithLevel(gl.LevelInfo)))

	resp, err := rc.R().Get("https://lifecell.ua/uk/malii-biznes-lifecell/servisi/ip-telefoniia/")
	assert.NoError(t, err)

	d, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body()))
	assert.NoError(t, err)

	s := ExcludeScripts(d.Selection)
	str, err := goquery.OuterHtml(s.Find(selector))
	assert.NoError(t, err)

	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	mstr, err := m.String("text/html", str)
	assert.NoError(t, err)

	testhelp_printDiffAsTable(str, mstr, true)
	t.Log(lu.JsonPrettyStr(GetAllLinksFromSel("https://lifecell.ua/", s)))
}

func testhelp_printDiffAsTable(before, after string, isAutoWrapText bool) {
	tw := tablewriter.NewWriter(os.Stdout)
	tw.SetRowLine(false)
	tw.SetBorder(true)
	tw.SetHeaderLine(true)
	tw.SetAutoWrapText(isAutoWrapText)
	tw.SetNoWhiteSpace(false)
	tw.SetHeader([]string{"BEFORE", "AFTER"})
	tw.Append([]string{before, after})
	tw.Render()
}

func Test2(t *testing.T) {
	u1 := "https://lifecell.ua/uk/malii-biznes-lifecell/servisi/ip-telefoniia/"

	m := minify.New()
	m.AddFunc("text/html", html.Minify)

	tests := []struct {
		name string
		url  string
	}{
		{"75104473", u1},
	}

	processSelectionFNLifecellUa := func(s *goquery.Selection) *goquery.Selection {
		sel := s.Find("div.page-content > div").Children().Filter("div.breadcrumb")
		newsel := s.NotSelection(sel)
		t.Log("processSelectionFNLifecellUa_SELNEW>>", lo.Must(goquery.OuterHtml(newsel)))
		return s.Find("div.page-content > div").Children().Not(".iot_widget").Not(".breadcrumb")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := gonet.NewRestyClient(gl.New(gl.WithOC(), gl.WithLevel(gl.LevelInfo)))

			resp, err := rc.R().Get(tt.url)
			assert.NoError(t, err)

			minhtml, err := m.Bytes("text/html", resp.Body())
			assert.NoError(t, err)

			err = os.WriteFile(tt.name+".min.txt", minhtml, 0666)
			assert.NoError(t, err)

			err = os.WriteFile(tt.name+".txt", resp.Body(), 0666)
			assert.NoError(t, err)

			d, err := goquery.NewDocumentFromReader(bytes.NewReader(minhtml))
			assert.NoError(t, err)

			f, err := os.OpenFile("111render.html", os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
			assert.NoError(t, err)
			defer f.Close()

			err = goquery.Render(f, FindBodyAndExcludeScript(d.Selection))
			assert.NoError(t, err)

			pwp := ParseWebPage(lu.GenerateCtxWithRid(), processSelectionFNLifecellUa(FindBodyAndExcludeScript(d.Selection)), true)
			t.Log("PARSED: ", pwp)
		})
	}
}

func Test_ParseWebPageByURLResty(t *testing.T) {
	urlDocsBizcell := "https://docs-bizcell.lifecell.ua/plugins/viewsource/viewpagesrc.action?pageId="

	for _, pageId := range []string{"75104473", "75104607"} {
		t.Run(pageId, func(t *testing.T) {
			res := ParseWebPageByURLResty(lu.GenerateCtxWithRid(), gl.Defult(), urlDocsBizcell+pageId, false, nil)
			os.WriteFile(pageId+".txt", []byte(res.TextContent), 0666)
		})
	}
}
