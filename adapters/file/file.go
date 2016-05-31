package file

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/acs/logspout/router"
)

const rotateSize = 20971520

func init() {
	router.AdapterFactories.Register(NewFileAdapter, "file")
}

func NewFileAdapter(route *router.Route) (router.LogAdapter, error) {
	tmplStr := "{{.Data}}\n"
	tmpl, err := template.New("file").Parse(tmplStr)
	if err != nil {
		return nil, err
	}

	client, err := docker.NewClient(
		getopt("DOCKER_HOST", "unix:///var/run/docker.sock"))
	if err != nil {
		return nil, err
	}

	events := make(chan *docker.APIEvents)
	err = client.AddEventListener(events)
	if err != nil {
		return nil, err
	}

	l := &listener{
		client: client,
		events: events,
	}

	files := make(map[string]*logFile)
	containers := make(map[string]*acsContainerInfo)

	adapter := &FileAdapter{
		route:      route,
		tmpl:       tmpl,
		files:      files,
		listener:   l,
		containers: containers,
	}

	runningContainers, err := client.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		log.Println("file:", err)
		return nil, err
	}

	for _, container := range runningContainers {
		containerID := container.ID
		adapter.save(containerID)
	}

	go adapter.listen()

	return adapter, nil
}

type logFile struct {
	dir        string
	name       string
	file       *os.File
	size       int
	rotateSize int
}

type listener struct {
	client *docker.Client
	events chan *docker.APIEvents
}

type acsContainerInfo struct {
	project string
	store   string
	name    string
}

type FileAdapter struct {
	route      *router.Route
	tmpl       *template.Template
	files      map[string]*logFile
	listener   *listener
	containers map[string]*acsContainerInfo
}

func getopt(name, dfault string) string {
	value := os.Getenv(name)
	if value == "" {
		value = dfault
	}
	return value
}

func (a *FileAdapter) listen() {
	for event := range a.listener.events {
		switch event.Status {
		case "destroy":
			go a.clean(event.ID)
		case "start", "restart":
			a.save(event.ID)
		}
	}
}

func (a *FileAdapter) save(containerID string) {
	container, err := a.listener.client.InspectContainer(containerID)
	if err != nil {
		log.Println("file: inspect container failed")
		return
	}

	store := ""
	for _, kv := range container.Config.Env {
		kvp := strings.SplitN(kv, "=", 2)
		if len(kvp) == 2 && kvp[0] == "ACSLOGSPOUT" {
			store = strings.ToLower(kvp[1])
		}
	}

	info := &acsContainerInfo{
		project: container.Config.Labels["com.docker.compose.project"],
		store:   store,
		name:    container.Name,
	}
	a.containers[containerID] = info
}

func (a *FileAdapter) clean(containerID string) {
	time.Sleep(30 * time.Second)
	if container, exists := a.containers[containerID]; !exists {
		log.Println("file: container to be cleaned is not saved")
		return
	} else {
		runningContainers, err := a.listener.client.ListContainers(docker.ListContainersOptions{})
		if err != nil {
			log.Println("file:", err)
			return
		}
		for _, apiContainer := range runningContainers {
			runningContainer, err := a.listener.client.InspectContainer(apiContainer.ID)
			if err != nil {
				log.Println("file: inspect container failed")
				return
			}
			name := runningContainer.Name
			if name == container.name {
				log.Println("file: found a same name container")
				return
			}
		}

		path := filepath.Join(a.route.Address, container.project, container.store, container.name)
		err = os.RemoveAll(path)
		if err != nil {
			log.Println("file: delete log file failed", err)
			return
		}
		log.Println("file: delete dir " + path + " successfully")

		delete(a.containers, containerID)
		log.Println("file: clean container info")

		if f, exists := a.files[containerID]; exists {
			err = f.file.Close()
			if err != nil {
				log.Println("file: close file failed when clean it")
			}
			delete(a.files, containerID)
			log.Println("file: clean log file info")
		}
	}
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func rotate(old *logFile) (new *logFile, err error) {
	defer old.file.Close()

	oldPath := old.dir + "/" + old.name
	newPath := old.dir + "/" + old.name + ".1"
	if isExist(newPath) {
		err = os.Remove(newPath)
		if err != nil {
			log.Println("file:", err)
			return nil, err
		}
	}

	err = os.Rename(oldPath, newPath)
	if err != nil {
		log.Println("file:", err)
		return nil, err
	}

	f, err := os.Create(oldPath)
	if err != nil {
		log.Println("file:", err)
		return nil, err
	}

	new = &logFile{file: f, size: 0, dir: old.dir, name: old.name, rotateSize: rotateSize}

	return new, nil
}

func (a *FileAdapter) Stream(logstream chan *router.Message) {
	var fileNow *logFile
	logName := "stdout"

	for message := range logstream {
		containerID := message.Container.ID

		if file, exists := a.files[containerID]; exists {
			fileNow = file
		} else {
			project := message.Container.Config.Labels["com.docker.compose.project"]
			if project == "" {
				log.Println("file: project name is null")
				continue
			}

			containerName := message.Container.Name

			store := ""
			for _, kv := range message.Container.Config.Env {
				kvp := strings.SplitN(kv, "=", 2)
				if len(kvp) == 2 && kvp[0] == "ACSLOGSPOUT" {
					store = strings.ToLower(kvp[1])
				}
			}

			logDir := filepath.Join(a.route.Address, project, store, containerName)
			if err := os.MkdirAll(logDir, 0777); err != nil {
				log.Println("file:", err)
				return
			}

			logPath := filepath.Join(logDir, logName)
			log.Println("file: Create a new file, file path is:", logPath)

			f, err := os.Create(logPath)
			if err != nil {
				log.Println("file:", err)
				return
			}

			file := logFile{file: f, size: 0, dir: logDir, name: logName, rotateSize: rotateSize}
			a.files[containerID] = &file
			fileNow = &file
		}

		buf := new(bytes.Buffer)
		err := a.tmpl.Execute(buf, message)
		if err != nil {
			log.Println("file:", err)
			return
		}

		if fileNow.size+buf.Len() > fileNow.rotateSize {
			log.Println("Rotate file:", fileNow.name)
			fileNow, err = rotate(fileNow)
			if err != nil {
				log.Println("file:", err)
				return
			}
			a.files[containerID] = fileNow
		}

		fileNow.size += buf.Len()

		//log.Println("debug:", buf.String())
		_, err = fileNow.file.Write(buf.Bytes())
		if err != nil {
			log.Println("file:", err)
			return
		}
	}
}
