package main

import (
	"fmt"
	"time"
	"net/http"
	"html/template"
	"database/sql"

	"github.com/gorilla/mux"
	"github.com/microcosm-cc/bluemonday"
)

var (
	dbFile string = "yarr.db"
	Db *sql.DB
	Ps *bluemonday.Policy
)

type Episode struct {
	Title string
	Description template.HTML
	Media string
	MediaType string
	Thumbnail string
	Id string
	Published time.Time

	PodId string
}

type Pod struct {
    Title string
    Description template.HTML
	AlbumArt string
	Creator string
	Categories []string
	Rss string
	Id string
	Added time.Time
	Link string

	Episodes []Episode
}

type MainPage struct {
    Pods []Pod
}

type PodPage struct {
	Episodes []Episode
	Podcast Pod
}

type EpisodePage struct {
	Epi Episode
}

func HandleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func InitBm() {
	Ps = bluemonday.NewPolicy()

	Ps.AllowStandardURLs()

	Ps.AllowAttrs("href").OnElements("a")
	Ps.AllowElements("p")
}

func webServer() {
	r := mux.NewRouter()

	episodeTmpl := template.Must(template.ParseFiles("templates/episode.html"))

	r.HandleFunc("/episode/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		episode, err := GetEpisode(id)
		if err != nil {
			fmt.Println(err)
			return
		}

		data := EpisodePage{
			Epi: episode,
		}

		err = episodeTmpl.Execute(w, data)
		if err != nil {
			fmt.Println(err)
			return
		}
	})

	podTmpl := template.Must(template.ParseFiles("templates/pod.html"))

	r.HandleFunc("/pods/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		episodes, err := GetAllEpisodes(id)
		if err != nil {
			fmt.Println(err)
			return
		}

		pod, err := GetPod(id)
		if err != nil {
			fmt.Println(err)
			return
		}

		data := PodPage{
            Episodes: episodes,
			Podcast: pod,
        }

		err = podTmpl.Execute(w, data)
		if err != nil {
			fmt.Println(err)
			return
		}
	})

	mainTmpl := template.Must(template.ParseFiles("templates/main.html"))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pods, err := TopPods(15)
		if err != nil {
			fmt.Println(err)
			return
		}

		data := MainPage{
            Pods: pods,
        }
        mainTmpl.Execute(w, data)
	})

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", r)
}

func main() {
	err := InitDb()
	HandleErr(err)

	InitBm()

	// err = ResetDb()
	// HandleErr(err)

	err = UpdateRss("https://feeds.fireside.fm/bibleinayear/rss")
	HandleErr(err)

	err = UpdateRss("https://rss.art19.com/apology-line")
	HandleErr(err)

	err = UpdateRss("https://access.acast.com/rss/5fc7c9db52d6971d13f1e77f/kzzzle7i")
	HandleErr(err)

	webServer()
}
