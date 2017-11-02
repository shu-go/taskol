package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "taskol"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "verbose", Usage: "verbose output to stderr"},
		cli.StringFlag{Name: "target", Usage: "target root directory"},
		cli.StringFlag{Name: "link", Usage: "link files directory"},
		cli.StringFlag{Name: "format", Value: ":tdate:_:pabb:_:tname:", Usage: "name format of each link file"},
		cli.StringFlag{Name: "ignores", Value: "!#@", Usage: "prefix characters"},
	}
	app.Action = func(c *cli.Context) error {
		targetDir := c.GlobalString("target")
		linkDir := c.GlobalString("link")
		linkFormat := c.GlobalString("format")
		ignores := c.GlobalString("ignores")

		if len(targetDir) == 0 {
			fmt.Fprintf(os.Stderr, "target directory is missing.\n")
			return fmt.Errorf("target directory is missing.")
		}
		if len(linkDir) == 0 {
			fmt.Fprintf(os.Stderr, "link directory is missing.\n")
			return fmt.Errorf("link directory is missing.")
		}
		if len(linkFormat) == 0 {
			fmt.Fprintf(os.Stderr, "link format is missing.\n")
			return fmt.Errorf("link format is missing.")
		}

		targetDir = strings.Replace(targetDir, `\`, `/`, -1)
		linkDir = strings.Replace(linkDir, `\`, `/`, -1)

		return run(targetDir, linkDir, linkFormat, ignores)
	}
	app.Run(os.Args)
}

func run(targetDir, linkDir, linkFormat, ignores string) error {
	for _, lnk := range listLinkFiles(linkDir, ignores) {
		if err := os.Remove(lnk); err != nil {
			println(err.Error())
		}
	}

	for _, p := range listProjectDirs(targetDir, ignores) {
		pabb, pname := projectAbbAndName(p)
		println(pabb, pname)

		for _, t := range listTaskDirs(p, ignores) {
			tname, tdate := taskNameAndDate(t)

			lnkName := linkName(linkFormat, pabb, pname, tname, tdate)
			println(t, "=>", lnkName)

			if err := createShortcut(t, linkDir+"/"+lnkName+".lnk"); err != nil {
				println(err.Error())
			}
		}
	}

	return nil
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
			m, _ := strconv.Atoi(subs[2])
			d, _ := strconv.Atoi(subs[3])
			dt := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
			date = &dt
		} else {
			name = c
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

	return filter(
		files,
		func(i int) bool { return !isDir(files[i]) },
		func(i int) bool { return !shouldBeIgnored(files[i], ignores) },
	)
}

func listProjectDirs(baseDir, ignores string) []string {
	dirs, err := filepath.Glob(baseDir + "/*") // Glob needs separators be /
	if err != nil {
		return nil
	}

	return filter(
		dirs,
		func(i int) bool { return isDir(dirs[i]) },
		func(i int) bool { return !shouldBeIgnored(dirs[i], ignores) },
	)
}

func listTaskDirs(prjDir, ignores string) []string {
	dirs, err := filepath.Glob(prjDir + "/t_*") // Glob needs separators be /
	if err != nil {
		return nil
	}

	return filter(
		dirs,
		func(i int) bool { return isDir(dirs[i]) },
		func(i int) bool { return !shouldBeIgnored(dirs[i], ignores) },
	)
}
