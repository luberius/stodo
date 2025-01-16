package todo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Priority int

const (
	None Priority = iota
	Low
	Medium
	High
)

func (p Priority) String() string {
	switch p {
	case Low:
		return "!"
	case Medium:
		return "!!"
	case High:
		return "!!!"
	default:
		return ""
	}
}

type Task struct {
	Text     string
	Done     bool
	Priority Priority
}

type Store struct {
	Tasks []Task
	path  string
}

func NewStore(filename string) *Store {
	stodoDir := ".stodo"
	if err := os.MkdirAll(stodoDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
	}

	path := filepath.Join(stodoDir, filename)

	if _, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644); err != nil {
		fmt.Printf("Error creating todo file: %v\n", err)
	}
	return &Store{path: path}
}

func (s *Store) Load() error {
	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var tasks []Task
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 4 {
			content := line[4:]
			priority := None
			if strings.HasPrefix(content, "!!! ") {
				priority = High
				content = content[4:]
			} else if strings.HasPrefix(content, "!! ") {
				priority = Medium
				content = content[3:]
			} else if strings.HasPrefix(content, "! ") {
				priority = Low
				content = content[2:]
			}
			tasks = append(tasks, Task{
				Text:     strings.TrimSpace(content),
				Done:     line[1] == 'x',
				Priority: priority,
			})
		}
	}
	s.Tasks = tasks
	return nil
}

func (s *Store) Save() error {
	file, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, task := range s.Tasks {
		mark := " "
		if task.Done {
			mark = "x"
		}
		priority := ""
		if task.Priority != None {
			priority = task.Priority.String() + " "
		}
		fmt.Fprintf(file, "[%s] %s%s\n", mark, priority, task.Text)
	}
	return nil
}

func (s *Store) Add(title string) {
	s.Tasks = append(s.Tasks, Task{
		Text:     title,
		Done:     false,
		Priority: None,
	})
	s.Save()
}

func (s *Store) Remove(index int) {
	if index >= 0 && index < len(s.Tasks) {
		s.Tasks = append(s.Tasks[:index], s.Tasks[index+1:]...)
	}

	s.Save()
}

func (s *Store) Toggle(index int) {
	if index >= 0 && index < len(s.Tasks) {
		s.Tasks[index].Done = !s.Tasks[index].Done
	}

	s.Save()
}

func (s *Store) CyclePriority(index int) {
	if index >= 0 && index < len(s.Tasks) {
		next := map[Priority]Priority{
			None:   Low,
			Low:    Medium,
			Medium: High,
			High:   None,
		}
		s.Tasks[index].Priority = next[s.Tasks[index].Priority]
	}

	s.Save()
}

func (s *Store) Archive(label string) error {
	timestamp := time.Now().Format("20060102150405")
	archiveName := filepath.Join(filepath.Dir(s.path), fmt.Sprintf("archive.%s", timestamp))
	if label != "" {
		archiveName = fmt.Sprintf("%s.%s", archiveName, label)
	}

	if err := os.Rename(s.path, archiveName); err != nil {
		return err
	}

	if _, err := os.Create(s.path); err != nil {
		return err
	}
	s.Tasks = []Task{}
	return nil
}
