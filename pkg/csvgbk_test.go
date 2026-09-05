package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestParseCsvDefGBKBytes(t *testing.T) {
	src := "渠道ID,备注,\"防沉迷(开1,关0,默认1)\",实名认证\r\n1,官网移动,1,1\r\n2,4399移动,0,0\r\n"
	raw, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if utf8.Valid(raw) {
		t.Fatal("fixture is valid UTF-8")
	}
	tab, err := ParseCsvDef(raw)
	if err != nil {
		t.Fatal(err)
	}
	if tab == nil {
		t.Fatal("nil table")
	}
	if got := strings.Join(tab.Header, "|"); got != "渠道ID|备注|防沉迷(开1,关0,默认1)|实名认证" {
		t.Fatalf("header %q", got)
	}
	if tab.Col("渠道ID") != 0 || tab.Col("备注") != 1 || tab.Col("防沉迷") != 2 || tab.Col("实名认证") != 3 {
		t.Fatalf("col index %d %d %d %d", tab.Col("渠道ID"), tab.Col("备注"), tab.Col("防沉迷"), tab.Col("实名认证"))
	}
	if len(tab.Rows) != 2 || len(tab.Rows[0]) != 4 || tab.Rows[0][1] != "官网移动" || tab.Rows[1][1] != "4399移动" {
		t.Fatalf("rows %q", tab.Rows)
	}
}

func TestParseCsvDefUTF8Unchanged(t *testing.T) {
	src := "ID,Name,ENName,Key\n1,石头,Stone,stone\n"
	tab, err := ParseCsvDef([]byte("\xEF\xBB\xBF" + src))
	if err != nil {
		t.Fatal(err)
	}
	if tab == nil || len(tab.Rows) != 1 || tab.Cell(tab.Rows[0], "Name") != "石头" {
		t.Fatalf("table %+v", tab)
	}
}

func TestParseCsvDefGBKTables(t *testing.T) {
	dir := testPkgDir(t)
	cases := []struct {
		pkgFile string
		table   string
		cols    []string
		minRows int
	}{
		{"game_script.pkg", "antiaddiction", []string{"渠道ID", "备注", "防沉迷"}, 60},
		{"game_script.pkg", "bperrorcode", []string{"ID", "Name", "Remarks"}, 5},
		{"game_script.pkg", "liteconfig", []string{"功能ID", "子ID", "功能名称", "子功能名称"}, 30},
		{"patch_game_script.pkg", "itemuseskindef", []string{"道具ID", "皮肤ID"}, 40},
	}
	for _, c := range cases {
		t.Run(c.table, func(t *testing.T) {
			path := filepath.Join(dir, c.pkgFile)
			if _, err := os.Stat(path); err != nil {
				t.Skip("no " + c.pkgFile)
			}
			rd, err := OpenOverlayFiles(path, "")
			if err != nil {
				t.Fatal(err)
			}
			_, raw, err := rd.LookupCsvDef(c.table)
			if err != nil {
				t.Fatal(err)
			}
			if utf8.Valid(raw) {
				t.Skip("table is UTF-8 in this pkg build")
			}
			tab, err := ParseCsvDef(raw)
			if err != nil {
				t.Fatal(err)
			}
			if tab == nil {
				t.Fatal("nil table")
			}
			for _, col := range c.cols {
				if tab.Col(col) < 0 {
					t.Errorf("column %q missing in header %q", col, tab.Header)
				}
			}
			if len(tab.Rows) < c.minRows {
				t.Errorf("rows=%d want>=%d", len(tab.Rows), c.minRows)
			}
			for i, row := range tab.Rows {
				if len(row) != len(tab.Header) {
					t.Errorf("row %d has %d fields, header %d", i, len(row), len(tab.Header))
				}
			}
		})
	}
}
