package file

import (
	"github.com/acs/logspout/router"
	"github.com/fsouza/go-dockerclient"
	"strconv"
	"testing"
	"time"
)

func TestStream(t *testing.T) {
	route := &router.Route{Adapter: "file", Address: "/tmp/test"}
	fileAdapter, err := NewFileAdapter(route)
	if err != nil {
		t.Error("NewFileAdapter failed")
	}

	logStream := make(chan *router.Message)

	go fileAdapter.Stream(logStream)

	go mockPump(logStream)

	time.Sleep(300 * 1e9)
}

func mockPump(logStream chan *router.Message) {
	for i := 0; i < 100; i++ {
		println("start to write")
		env := []string{"ACSLOGSPOUT=container"}

		con := docker.Config{Env: env}
		con.Labels = make(map[string]string)
		con.Labels["com.docker.compose.project"] = "logspout_project"
		con.Labels["com.docker.compose.service"] = "logspout_service"

		d := docker.Container{Config: &con}

		d.Name = "container"

		m := router.Message{}
		m.Container = &d
		m.Data = "test stream and logrotate:" + strconv.Itoa(i)
		m.Source = "stdout"

		logStream <- &m

		time.Sleep(2 * 1e9)
	}
}
