package main

import (
	"fmt"
	"time"
	// "errors"
	"strings"
	"html/template"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func ResetDb() error {
	statements := []string{
		"drop table if exists podcasts;",
		"drop table if exists episodes;",
		"drop table if exists podcasts_fts;",
		"drop table if exists episodes_fts;",
		"drop trigger if exists podcasts_fts_insert",
		"drop trigger if exists podcasts_fts_update",
		"drop trigger if exists podcasts_fts_delete",
		"drop trigger if exists episodes_fts_insert",
		"drop trigger if exists episodes_fts_update",
		"drop trigger if exists episodes_fts_delete",
		"create table podcasts (id text not null primary key, title text, description text, albumart text, creator text, categories text, rss text, added datetime, link text, approved bool);",
		"create table episodes (id text not null primary key, podId text, title text, description text, thumbnail text, media text, mediaType text, published datetime);",
		"create virtual table podcasts_fts using fts5 (title, description, content=podcasts);",
		"create virtual table episodes_fts using fts5 (title, description, content=episodes);",
		`CREATE TRIGGER podcasts_fts_insert AFTER INSERT ON podcasts BEGIN 
			INSERT INTO podcasts_fts (rowid, title, description) VALUES (new.rowid, new.title, new.description); 
		END;`,
		`CREATE TRIGGER podcasts_fts_delete AFTER DELETE ON podcasts
		BEGIN
			INSERT INTO podcasts_fts (podcasts_fts, rowid, title, description) VALUES ('delete', old.rowid, old.title, old.description);
		END;`,
		`CREATE TRIGGER podcasts_fts_update AFTER UPDATE ON podcasts
		BEGIN
			INSERT INTO podcasts_fts (podcasts_fts, rowid, title, description) VALUES ('delete', old.rowid, old.title, old.description);
			INSERT INTO podcasts_fts (rowid, title, description) VALUES (new.rowid, new.title, new.description);
		END;`,
		`CREATE TRIGGER episodes_fts_insert AFTER INSERT ON episodes BEGIN 
			INSERT INTO episodes_fts (rowid, title, description) VALUES (new.rowid, new.title, new.description); 
		END;`,
		`CREATE TRIGGER episodes_fts_delete AFTER DELETE ON episodes
		BEGIN
			INSERT INTO episodes_fts (episodes_fts, rowid, title, description) VALUES ('delete', old.rowid, old.title, old.description);
		END;`,
		`CREATE TRIGGER episodes_fts_update AFTER UPDATE ON episodes
		BEGIN
			INSERT INTO episodes_fts (episodes_fts, rowid, title, description) VALUES ('delete', old.rowid, old.title, old.description);
			INSERT INTO episodes_fts (rowid, title, description) VALUES (new.rowid, new.title, new.description);
		END;`,
	}

	for _, stmt := range statements {
		_, err := Db.Exec(stmt)
		if err != nil {
			return err
		}
	}

	return nil
}

func InitDb() error {
	err := fmt.Errorf("Failed to open database.")
	Db, err = sql.Open("sqlite3", dbFile)
	return err
}

func SearchEpisodes(podId string, keywords string) ([]Episode, error) {
	stmt, err := Db.Prepare("select * from episodes where rowid in (select rowid from episodes_fts where episodes_fts match ? order by rank) and podId=?")

	if err != nil {
		return []Episode{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(keywords, podId)
	if err != nil {
		return []Episode{}, err
	}

	return GetEpisodes(rows)
}

func GetAllApprovedPods() ([]Pod, error) {
	stmt, err := Db.Prepare("select * from podcasts where approved=1 order by added")

	if err != nil {
		return []Pod{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return []Pod{}, err
	}

	return GetPods(rows)
}

func GetAllEpisodes(podId string) ([]Episode, error) {
	stmt, err := Db.Prepare("select * from episodes where podId=? order by published desc")

	if err != nil {
		return []Episode{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(podId)
	if err != nil {
		return []Episode{}, err
	}

	return GetEpisodes(rows)
}

func GetEpisode(id string) (Episode, error) {
	stmt, err := Db.Prepare("select * from episodes where id=?")
	if err != nil {
		return Episode{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		return Episode{}, err
	}

	episodes, err := GetEpisodes(rows)
	if err != nil {
		return Episode{}, err
	}

	return episodes[0], err
}

func GetEpisodes(rows *sql.Rows) ([]Episode, error) {
	defer rows.Close()

	episodes := []Episode{}
	for rows.Next() {
		var id string
		var podId string
		var title string
		var description string
		var thumbnail string
		var media string
		var mediaType string
		var published time.Time

		err := rows.Scan(&id, &podId, &title, &description, &thumbnail, &media, &mediaType, &published)
		if err != nil {
			return []Episode{}, err
		}

		episode := Episode{
			Title: title,
			Description: template.HTML(description),
			Media: media,
			MediaType: mediaType,
			Thumbnail: thumbnail,
			Id: id,
			Published: published,
			PodId: podId,
		}

		episodes = append(episodes, episode)
	}

	err := rows.Err()
	if err != nil {
		return []Episode{}, err
	}

	return episodes, nil
}

func TopPods(num int, admin bool) ([]Pod, error) {
	var stmt *sql.Stmt
	var err error
	if !admin {
		stmt, err = Db.Prepare("select * from podcasts where approved=1 order by added desc limit ?")
	} else {
		stmt, err = Db.Prepare("select * from podcasts order by added desc limit ?")
	}

	if err != nil {
		return []Pod{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(num)
	if err != nil {
		return []Pod{}, err
	}

	return GetPods(rows)
}

func GetPod(id string) (Pod, error) {
	stmt, err := Db.Prepare("select * from podcasts where id=?")

	if err != nil {
		return Pod{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		return Pod{}, err
	}

	pods, err := GetPods(rows)
	if err != nil {
		return Pod{}, err
	}

	return pods[0], err
}

func GetPods(rows *sql.Rows) ([]Pod, error) {
	defer rows.Close()

	pods := []Pod{}
	for rows.Next() {
		var id string
		var title string
		var description string
		var albumart string
		var creator string
		var cats string
		var categories []string
		var rss string
		var link string 
		var added time.Time
		var approved bool

		err := rows.Scan(&id, &title, &description, &albumart, &creator, &cats, &rss, &added, &link, &approved)
		if err != nil {
			return []Pod{}, err
		}

		if !approved {
			fmt.Println("Unapproved")
		}

		categories = strings.Split(cats, ",")

		podcast := Pod{
			Title: title,
			Description: template.HTML(description),
			AlbumArt: albumart,
			Creator: creator,
			Categories: categories,
			Rss: rss,
			Id: id,
			Added: added,
			Link: link,
		}

		pods = append(pods, podcast)
	}

	err := rows.Err()
	if err != nil {
		return []Pod{}, err
	}

	return pods, nil
}

func SearchPods(keywords string) ([]Pod, error) {
	stmt, err := Db.Prepare("select * from podcasts where rowid in (select rowid from podcasts_fts where podcasts_fts match ? order by rank) and approved=1")

	if err != nil {
		return []Pod{}, err
	}

	defer stmt.Close()

	rows, err := stmt.Query(keywords)
	if err != nil {
		return []Pod{}, err
	}

	return GetPods(rows)
}

func AddPod(pod Pod) error {
	tx, err := Db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("insert into podcasts(id, title, description, albumart, creator, categories, rss, added, link, approved) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	cat := strings.Join(pod.Categories, ",")

	defer stmt.Close()
	_, err = stmt.Exec(pod.Id, pod.Title, pod.Description, pod.AlbumArt, pod.Creator, cat, pod.Rss, pod.Added, pod.Link, false)
	if err != nil {
		return err
	}
	
	tx.Commit()

	return nil
}

func ApprovePod(id string) error {
	tx, err := Db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("update podcasts set approved=1 where id=?")
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}
	
	tx.Commit()

	return nil
}

func DeletePod(id string) error {
	tx, err := Db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("delete from podcasts where id=?")
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}
	
	tx.Commit()

	return nil
}

func AddEpisode(episode Episode) error {
	tx, err := Db.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare("insert into episodes(id, podId, title, description, thumbnail, media, mediaType, published) values(?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(episode.Id, episode.PodId, episode.Title, episode.Description, episode.Thumbnail, episode.Media, episode.MediaType, episode.Published)
	if err != nil {
		return err
	}
	
	tx.Commit()

	return nil
}

func PodExists(id string) (bool, error) {
	rows, err := Db.Query("select id from podcasts")
	if err != nil {
		return false, err
	}

	defer rows.Close()
	for rows.Next() {
		var nid string
		err = rows.Scan(&nid)
		if err != nil {
			return false, err
		}

		if id == nid {
			return true, err
		}
	}

	err = rows.Err()
	if err != nil {
		return false, err
	}

	return false, err
}