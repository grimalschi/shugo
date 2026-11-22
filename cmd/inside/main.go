// Command inside walks a small project through every stage of Shugo and
// writes what lies between the stages into one HTML page: the sources, the
// JavaScript and the skeleton the compiler makes of them, the data database
// with its journal, the conversation of shugo with the JavaScript side, the
// records of the store and how they lead to the data, and what every update
// changes. It is a stand for looking inside, not a part of the framework.
//
//	go build -o bin/shugo ./cmd/shugo
//	go run ./cmd/inside [-shugo bin/shugo] [examples/inside]
//
// The page is written to report/index.html in the project.
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	shugo := flag.String("shugo", "bin/shugo", "the shugo command to run")
	flag.Parse()
	project := "examples/inside"
	if flag.NArg() > 0 {
		project = flag.Arg(0)
	}
	if err := walkThrough(project, *shugo); err != nil {
		fmt.Fprintln(os.Stderr, "inside:", err)
		os.Exit(1)
	}
}

// stepPlan is one change of the data, as data/steps.json describes it.
type stepPlan struct {
	Title string   `json:"title"`
	Why   string   `json:"why"`
	SQL   []string `json:"sql"`
}

func walkThrough(project, shugo string) error {
	var err error
	if project, err = filepath.Abs(project); err != nil {
		return err
	}
	if shugo, err = filepath.Abs(shugo); err != nil {
		return err
	}
	out := filepath.Join(project, "report")
	for _, gone := range []string{"_build", "report", "data/site.db", "data/site.db-wal", "data/site.db-shm"} {
		if err := os.RemoveAll(filepath.Join(project, gone)); err != nil {
			return err
		}
	}
	seed, err := os.ReadFile(filepath.Join(project, "data", "seed.sql"))
	if err != nil {
		return err
	}
	if err := execute(project, string(seed)); err != nil {
		return fmt.Errorf("the data of the project: %w", err)
	}

	r := &report{Project: project, Seed: string(seed)}
	run := func(command, talk string) (string, error) {
		cmd := exec.Command(shugo, command, "-json", "-runtime", filepath.Join(project, "tap.sh"), project)
		cmd.Env = append(os.Environ(), "SHUGO_TAP="+filepath.Join(out, "talk", talk))
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("shugo %s: %v\n%s", command, err, stderr.String())
		}
		return stdout.String(), nil
	}

	// The first build.
	built, err := run("build", "build")
	if err != nil {
		return err
	}
	if r.Compiled, err = lookAtCompiled(project); err != nil {
		return err
	}
	if r.Files, err = sources(project, r.Compiled); err != nil {
		return err
	}
	now, err := lookAt(project)
	if err != nil {
		return err
	}
	r.Build = stage{Report: pretty(built), Look: now}
	if r.Build.Requests, r.Build.Answers, err = talk(filepath.Join(out, "talk", "build")); err != nil {
		return err
	}
	r.Build.Rounds = rounds(r.Build.Requests, r.Build.Answers)
	r.Story = story(r.Compiled, now, "/products/jug")

	// The changes of the data, one update after each.
	plans, err := os.ReadFile(filepath.Join(project, "data", "steps.json"))
	if err != nil {
		return err
	}
	var steps []stepPlan
	if err := json.Unmarshal(plans, &steps); err != nil {
		return fmt.Errorf("data/steps.json: %w", err)
	}
	for i, plan := range steps {
		before := now
		if err := execute(project, strings.Join(plan.SQL, ";\n")); err != nil {
			return fmt.Errorf("%s: %w", plan.Title, err)
		}
		pending, err := lookAtData(filepath.Join(project, "data", "site.db"))
		if err != nil {
			return err
		}
		position := dataPosition(before.Store)
		found, err := changes(pending, position, project)
		if err != nil {
			return err
		}
		s := step{Number: i + 1, Plan: plan, Position: position}
		for _, line := range pending.Journal {
			if line.Seq > position {
				s.Journal = append(s.Journal, line)
			}
		}
		for _, c := range found {
			hits, misses := reach(before.Store, c)
			s.Changes = append(s.Changes, changeView{change: c, Hits: hits, Misses: misses})
		}
		updated, err := run("update", fmt.Sprintf("step-%d", i+1))
		if err != nil {
			return err
		}
		if now, err = lookAt(project); err != nil {
			return err
		}
		s.Report = pretty(updated)
		json.Unmarshal([]byte(updated), &s.Summary)
		s.Diff = diff(before.Store, now.Store)
		if since, err := strconv.ParseInt(s.Diff.Version[0], 10, 64); err == nil {
			if s.Diff.Build, s.Diff.Remove, err = exportAfter(filepath.Join(project, "_build", "store.db"), since); err != nil {
				return err
			}
		}
		s.HTML = pagesThatDiffer(before.HTML, now.HTML)
		if s.Requests, s.Answers, err = talk(filepath.Join(out, "talk", fmt.Sprintf("step-%d", i+1))); err != nil {
			return err
		}
		s.Rounds = rounds(s.Requests, s.Answers)
		s.After = now
		r.Steps = append(r.Steps, s)
	}
	r.Final = now
	r.Serving = serve(r.Build.Look, "/products/jug")

	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	page, err := r.render()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "index.html"), page, 0o644); err != nil {
		return err
	}
	fmt.Printf("written %s, %d KB\n", filepath.Join(out, "index.html"), len(page)>>10)
	return nil
}

// execute runs statements against the data database of the project, the
// way any program that brings data would.
func execute(project, statements string) error {
	db, err := sql.Open("sqlite", "file:"+filepath.Join(project, "data", "site.db"))
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(statements)
	return err
}

// dataPosition is the line of the journal the store has dealt with.
func dataPosition(s storeLook) int64 {
	for _, p := range s.Meta {
		if p.Name == "data" {
			var position int64
			fields := strings.Fields(p.Value)
			if len(fields) > 0 {
				fmt.Sscan(fields[len(fields)-1], &position)
			}
			return position
		}
	}
	return 0
}

func talk(folder string) (requests, answers []frame, err error) {
	if requests, err = frames(filepath.Join(folder, "requests.bin"), fromGo); err != nil {
		return nil, nil, err
	}
	answers, err = frames(filepath.Join(folder, "answers.bin"), fromJS)
	return requests, answers, err
}

// sources reads the files of the project, with what the compiler made of
// every .shugo file.
func sources(project string, c *compiled) ([]sourceFile, error) {
	var out []sourceFile
	err := filepath.WalkDir(project, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(project, path)
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if rel == "_build" || rel == "report" || rel == "data" || strings.HasPrefix(entry.Name(), ".") && rel != "." {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".shugo") && !strings.HasSuffix(rel, ".ts") && rel != "shugo.json" {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		f := sourceFile{Path: rel, Source: string(source), Module: c.Modules[rel], Number: -1}
		for i, file := range c.Site.Files {
			if file.Path == rel {
				f.Number, f.Shared = i, file.Shared
				for p, ops := range file.Parts {
					f.Parts = append(f.Parts, partView{Number: p, Ops: listOps(ops, 0)})
				}
			}
		}
		out = append(out, f)
		return nil
	})
	sort.SliceStable(out, func(a, b int) bool { return order(out[a].Path) < order(out[b].Path) })
	return out, err
}

// order puts the files in the order they are best read in.
func order(path string) string {
	switch {
	case path == "shugo.json":
		return "0"
	case strings.HasPrefix(path, "src/pages/"):
		return "1" + path
	case strings.HasPrefix(path, "src/components/"):
		return "2" + path
	}
	return "3" + path
}
