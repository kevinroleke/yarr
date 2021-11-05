package main

import (
	"os"
	"fmt"
	"time"
	"net/http"
	"html/template"
	"database/sql"

	"github.com/gorilla/mux"
	"github.com/kataras/hcaptcha"
	"github.com/microcosm-cc/bluemonday"
)

var (
	dbFile string = "yarr.db"
	Db *sql.DB
	Ps *bluemonday.Policy
	siteKey string = os.Getenv("HCAPTCHA_SITE_KEY")
	secretKey string = os.Getenv("HCAPTCHA_SECRET_KEY")
	adminKey string = os.Getenv("ADMIN_KEY")
	errmsg string = "there was an error D:"
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
	Admin bool
	AdminKey string
}

type PodPage struct {
	Episodes []Episode
	Podcast Pod
	SiteKey string
}

type EpisodePage struct {
	Epi Episode
}

type AddPage struct {
	SiteKey string
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

	hclient := hcaptcha.New(secretKey)
	addTmpl := template.Must(template.ParseFiles("templates/add.html"))

	r.HandleFunc("/add/", func(w http.ResponseWriter, r *http.Request) {
		data := AddPage{
            SiteKey: siteKey,
        }
        addTmpl.Execute(w, data)
	})

	r.HandleFunc("/admin/approve/{id}/", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		admin := r.URL.Query().Get("admin")
		if admin != adminKey {
			fmt.Fprintf(w, "u r not authenticated whomp")
			return
		}

		err := ApprovePod(id)
		if err != nil {
			fmt.Fprintf(w, "FAILED TO APPROVE %s", id)
			return
		}

		fmt.Fprintf(w, "APPROVED %s", id)
	})

	r.HandleFunc("/admin/delete/{id}/", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		admin := r.URL.Query().Get("admin")
		if admin != adminKey {
			fmt.Fprintf(w, "u r not authenticated whomp")
			return
		}

		err := DeletePod(id)
		if err != nil {
			fmt.Fprintf(w, "FAILED TO DELETE %s", id)
			return
		}

		fmt.Fprintf(w, "DELETED %s", id)
	})

	r.HandleFunc("/add/podcast", hclient.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := hcaptcha.Get(r)
		if !ok {
			fmt.Fprintf(w, "you failed da captcha")
			return
		}

		r.ParseForm()
		rssLink := r.FormValue("rssLink")

		err := UpdateRss(rssLink)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		fmt.Fprintf(w, "submitted successfully")
	}))

	r.HandleFunc("/pods/{id}/refresh", hclient.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, ok := hcaptcha.Get(r)
		if !ok {
			fmt.Fprintf(w, "you failed da captcha")
			return
		}

		// get rss
		pod, err := GetPod(id)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		err = UpdateRss(pod.Rss)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		http.Redirect(w, r, "/pods/" + id, http.StatusTemporaryRedirect)
	}))

	episodeTmpl := template.Must(template.ParseFiles("templates/episode.html"))

	r.HandleFunc("/episode/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		episode, err := GetEpisode(id)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		data := EpisodePage{
			Epi: episode,
		}

		err = episodeTmpl.Execute(w, data)
		if err != nil {
			fmt.Fprintf(w, errmsg)
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
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		pod, err := GetPod(id)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		data := PodPage{
            Episodes: episodes,
			Podcast: pod,
			SiteKey: siteKey,
        }

		err = podTmpl.Execute(w, data)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}
	})

	r.HandleFunc("/search/episodes/{podId}/{keywords}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		podId := vars["podId"]
		keywords := vars["keywords"]

		episodes, err := SearchEpisodes(podId, keywords)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		pod, err := GetPod(podId)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		data := PodPage{
            Episodes: episodes,
			Podcast: pod,
        }

		err = podTmpl.Execute(w, data)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}
	})

	mainTmpl := template.Must(template.ParseFiles("templates/main.html"))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		admin := r.URL.Query().Get("admin")
		adminView := false
		if admin == adminKey {
			adminView = true
		}

		pods, err := TopPods(15, adminView)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		data := MainPage{
            Pods: pods,
			Admin: adminView,
			AdminKey: admin,
        }
        err = mainTmpl.Execute(w, data)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}
	})

	r.HandleFunc("/search/pods/{keywords}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		keywords := vars["keywords"]

		pods, err := SearchPods(keywords)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}

		data := MainPage{
            Pods: pods,
        }
        err = mainTmpl.Execute(w, data)
		if err != nil {
			fmt.Fprintf(w, errmsg)
			fmt.Println(err)
			return
		}
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

	// err = UpdateRss("https://feeds.fireside.fm/bibleinayear/rss")
	// HandleErr(err)

	// err = UpdateRss("https://rss.art19.com/apology-line")
	// HandleErr(err)

	// err = UpdateRss("https://access.acast.com/rss/5fc7c9db52d6971d13f1e77f/kzzzle7i")
	// HandleErr(err)

	go EveryHour(UpdateAll)

	webServer()
}
