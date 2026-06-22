package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kartikkabadi/go-learn/internal/store"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(root, "progress", "go-learn.db")
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	bundleDir := filepath.Join(root, "content-bundles")
	entries, err := os.ReadDir(bundleDir)
	if err != nil {
		log.Fatal(err)
	}

	var lessonFiles []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			lessonFiles = append(lessonFiles, name)
		}
	}
	sort.Strings(lessonFiles)

	for _, name := range lessonFiles {
		path := filepath.Join(bundleDir, name)
		b, err := store.LoadContentBundle(path)
		if err != nil {
			log.Fatalf("load %s: %v", name, err)
		}
		if err := st.ImportBundle(b); err != nil {
			log.Fatalf("import %s: %v", name, err)
		}
		fmt.Printf("imported lesson %s\n", b.Lesson.ID)
	}

	importJSON := func(filename string, importFn func([]byte) error) {
		data, err := os.ReadFile(filepath.Join(bundleDir, filename))
		if err != nil {
			log.Fatalf("read %s: %v", filename, err)
		}
		if err := importFn(data); err != nil {
			log.Fatalf("import %s: %v", filename, err)
		}
		fmt.Printf("imported %s\n", filename)
	}

	importJSON("mission.json", func(data []byte) error {
		var b store.MissionBundle
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		return st.ImportMission(b)
	})

	importJSON("glossary.json", func(data []byte) error {
		var b store.GlossaryBundle
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		return st.ImportGlossary(b)
	})

	importJSON("insights.json", func(data []byte) error {
		var b store.InsightsBundle
		if err := json.Unmarshal(data, &b); err != nil {
			return err
		}
		return st.ImportInsights(b)
	})

	fmt.Println("import complete")
}
