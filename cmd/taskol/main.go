package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/shu/clise"
	"bitbucket.org/shu/gli"
)

type globalCmd struct {
	Verbose bool   `help:"標準エラーに追加情報を出力します"`
	Target  string `help:"対象のルートディレクトリーを指定します"`
	Link    string `help:"ショートカットを作成するディレクトリーを指定します"`
	Format  string `default:":tdate:_:pabb:_:tname:"  help:"作成されるショートカットのファイル名規則を指定します"`
	Ignores string `default:"!#@"  help:"走査対象外のディレクトリー名のプレフィックスを指定します"`
}

func (c globalCmd) Before() error {
	if len(c.Target) == 0 {
		return fmt.Errorf("対象のディレクトリが指定されていません --target")
	}
	if len(c.Link) == 0 {
		return fmt.Errorf("ショートカット作成先のディレクトリが指定されていません --link")
	}
	if len(c.Format) == 0 {
		return fmt.Errorf("ショートカットのファイル名規則が空白になっています --format")
	}

	return nil
}

func (c globalCmd) Run() error {
	targetDir := strings.Replace(c.Target, `\`, `/`, -1)
	linkDir := strings.Replace(c.Link, `\`, `/`, -1)

	// 既存のショートカットを全て削除。後で作り直すので。
	for _, lnk := range listLinkFiles(linkDir, c.Ignores) {
		if err := os.Remove(lnk); err != nil {
			println(err.Error())
		}
	}

	// ショートカット作成対象を格納しているディレクトリ
	for _, p := range listProjectDirs(targetDir, c.Ignores) {
		pabb, pname := projectAbbAndName(p)
		println(pabb, pname)

		for _, t := range listTaskDirs(p, c.Ignores) {
			tname, tdate := taskNameAndDate(t)

			lnkName := linkName(c.Format, pabb, pname, tname, tdate)
			println(t, "=>", lnkName)

			if err := createShortcut(t, linkDir+"/"+lnkName+".lnk"); err != nil {
				println(err.Error())
			}
		}
	}

	return nil
}

func main() {
	app := gli.New(&globalCmd{})
	app.Name = "taskol"
	app.Version = "0.2.0"
	app.Copyright = "(C) 2018 Shuhei Kubota"
	app.Desc = "仕掛中の作業フォルダへのショートカットを作るツール"
	app.Usage = `taskol --target=/path/to/work/root --link=/path/to/link/root`
	err := app.Run(os.Args)
	if err != nil {
		os.Exit(-1)
	}
}

func linkName(linkFormat, pabb, pname, tname string, tdate *time.Time) string {
	result := linkFormat
	result = strings.Replace(result, ":pabb:", pabb, -1)
	result = strings.Replace(result, ":pname:", pname, -1)
	result = strings.Replace(result, ":tname:", tname, -1)
	if tdate == nil {
		result = strings.Replace(result, ":tdate:", "", -1)
		result = strings.Replace(result, ":tdate-:", "", -1)
		result = strings.Replace(result, ":tdate年月日:", "", -1)
	} else {
		result = strings.Replace(result, ":tdate:", fmt.Sprintf("%04d%02d%02d", tdate.Year(), tdate.Month(), tdate.Day()), -1)
		result = strings.Replace(result, ":tdate-:", fmt.Sprintf("%04d-%02d-%02d", tdate.Year(), tdate.Month(), tdate.Day()), -1)
		result = strings.Replace(result, ":tdate年月日:", fmt.Sprintf("%04d年%02d月%02d日", tdate.Year(), tdate.Month(), tdate.Day()), -1)
	}
	return result
}

func taskNameAndDate(dir string) (name string, date *time.Time) {
	compoSepPtn := regexp.MustCompile(`_|\(|\)`)
	datePtn := regexp.MustCompile(`(\d{2,4})-?(\d{2})-?(\d{2})`)

	base := filepath.Base(dir)
	var withoutPrefix string
	if strings.HasPrefix(base, "t_") {
		withoutPrefix = base[2:len(base)]
	} else {
		withoutPrefix = base
	}

	compos := compoSepPtn.Split(withoutPrefix, -1)
	for _, c := range compos {
		if len(c) == 0 {
			continue
		}

		if subs := datePtn.FindStringSubmatch(c); len(subs) >= 4 {
			y, _ := strconv.Atoi(subs[1])
			if y < 100 {
				y += 2000
			}
			m, _ := strconv.Atoi(subs[2])
			d, _ := strconv.Atoi(subs[3])
			dt := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
			date = &dt
		} else {
			if len(name) != 0 {
				name += "_"
			}
			name += c
		}
	}

	return name, date
}

func projectAbbAndName(dir string) (abb, name string) {
	compoSepPtn := regexp.MustCompile(`_|\(|\)`)
	abbPtn := regexp.MustCompile(`[[:alnum:]]`)

	base := filepath.Base(dir)

	compos := compoSepPtn.Split(base, -1)
	for _, c := range compos {
		if len(c) == 0 {
			continue
		}

		if abbPtn.MatchString(c) {
			abb = c
			if len(name) == 0 {
				name = c
			}
		} else {
			if len(abb) == 0 {
				abb = c
			}
			name = c
		}
	}

	return abb, name
}

func isDir(p string) bool {
	info, err := os.Lstat(p)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func shouldBeIgnored(p, ignores string) bool {
	base := filepath.Base(p)
	// compare first runes
	for _, b := range base {
		for _, c := range ignores {
			if b == c {
				return true
			}
		}
		break
	}
	return false
}

func listLinkFiles(baseDir, ignores string) []string {
	files, err := filepath.Glob(baseDir + "/*.lnk") // Glob needs separators be /
	if err != nil {
		return nil
	}

	clise.Filter(
		&files,
		func(i int) bool { return !isDir(files[i]) },
		func(i int) bool { return !shouldBeIgnored(files[i], ignores) },
	)
	return files
}

func listProjectDirs(baseDir, ignores string) []string {
	dirs, err := filepath.Glob(baseDir + "/*") // Glob needs separators be /
	if err != nil {
		return nil
	}

	clise.Filter(
		&dirs,
		func(i int) bool { return isDir(dirs[i]) },
		func(i int) bool { return !shouldBeIgnored(dirs[i], ignores) },
	)
	return dirs
}

func listTaskDirs(prjDir, ignores string) []string {
	dirs, err := filepath.Glob(prjDir + "/t_*") // Glob needs separators be /
	if err != nil {
		return nil
	}

	clise.Filter(
		&dirs,
		func(i int) bool { return isDir(dirs[i]) },
		func(i int) bool { return !shouldBeIgnored(dirs[i], ignores) },
	)
	return dirs
}
